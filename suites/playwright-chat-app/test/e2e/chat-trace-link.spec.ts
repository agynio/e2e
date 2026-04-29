import { randomUUID } from 'node:crypto';
import { expect, type Page } from '@playwright/test';
import { test } from './fixtures';
import {
  createChat,
  getMessages,
  resolveIdentityId,
  sendChatMessage,
  setupTestAgent,
  waitForAgentReply,
} from './chat-api';
import { setSelectedOrganization } from './organization-helpers';
import { completeOidcLogin } from './sign-in-helper';

const CODEX_TEST_LLM_ENDPOINT =
  process.env.E2E_TEST_LLM_ENDPOINT ?? 'https://testllm.dev/v1/org/agynio/suite/codex/responses';
const CLAUDE_TEST_LLM_ENDPOINT =
  process.env.E2E_TEST_LLM_ENDPOINT_CLAUDE ?? 'https://testllm.dev/v1/org/agynio/suite/claude/messages';
const CLAUDE_INIT_IMAGE = process.env.CLAUDE_INIT_IMAGE ?? 'ghcr.io/agynio/agent-init-claude:latest';
const CLAUDE_PROTOCOL = 'PROTOCOL_ANTHROPIC_MESSAGES';

type TraceScenario = {
  name: string;
  endpoint: string;
  protocol?: string;
  initImage?: string;
};

const TRACE_SCENARIOS: TraceScenario[] = [
  {
    name: 'codex',
    endpoint: CODEX_TEST_LLM_ENDPOINT,
    initImage: process.env.E2E_AGENT_INIT_IMAGE,
  },
  {
    name: 'claude',
    endpoint: CLAUDE_TEST_LLM_ENDPOINT,
    protocol: CLAUDE_PROTOCOL,
    initImage: CLAUDE_INIT_IMAGE,
  },
];

async function openTraceFromChat(
  page: Page,
  params: { chatId: string; organizationId: string; messageId: string; messageText: string },
): Promise<void> {
  const chatLoaded = page.waitForResponse(
    (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
    { timeout: 15000 },
  );
  await page.goto(`/chats/${encodeURIComponent(params.chatId)}`);
  await chatLoaded;

  const messageRow = page.getByTestId('chat-message').filter({ hasText: params.messageText }).first();
  await expect(messageRow).toBeVisible({ timeout: 60000 });
  await messageRow.hover();

  const actionsTrigger = messageRow.getByTestId('message-actions-trigger');
  await expect(actionsTrigger).toBeVisible({ timeout: 10000 });
  await actionsTrigger.click();

  const traceLink = page.getByTestId('message-trace-link');
  await expect(traceLink).toBeVisible({ timeout: 10000 });

  const traceHref = await traceLink.getAttribute('href');
  if (!traceHref) {
    throw new Error('Trace link is missing href.');
  }

  const traceUrl = new URL(traceHref, page.url());
  expect(traceUrl.pathname).toBe(`/message/${params.messageId}`);
  expect(traceUrl.searchParams.get('orgId')).toBe(params.organizationId);

  const [tracePage] = await Promise.all([
    page.waitForEvent('popup'),
    traceLink.click(),
  ]);

  await tracePage.waitForLoadState('domcontentloaded');

  const callbackPromise = tracePage.waitForURL(/\/callback/, { timeout: 60000 }).catch(() => null);
  const completed = await completeOidcLogin(tracePage, { timeoutMs: 10000 });
  if (completed) {
    await callbackPromise;
  }

  const runUrlPattern = new RegExp(`/${params.organizationId}/runs/[0-9a-f]{32}(\\?.*)?$`);
  await expect(tracePage).toHaveURL(runUrlPattern, { timeout: 120000 });

  await expect(tracePage.getByTestId('run-summary-status')).toContainText(/finished/i, { timeout: 120000 });

  const eventsList = tracePage.getByTestId('run-events-list');
  await expect(eventsList).toBeVisible({ timeout: 120000 });
  const eventItems = eventsList.locator('[data-testid^="run-event-"]');
  await expect.poll(() => eventItems.count(), { timeout: 120000 }).toBeGreaterThanOrEqual(2);

  const messageEvent = eventsList.getByRole('button', { name: /Message • Source/ }).first();
  await messageEvent.click();
  await expect(tracePage.getByTestId('run-event-details-message-content')).toContainText(params.messageText);

  const llmEvents = eventsList.getByRole('button', { name: /LLM Call/ });
  await expect.poll(() => llmEvents.count(), { timeout: 120000 }).toBeGreaterThan(0);
  await llmEvents.first().click();
  await expect(tracePage.getByTestId('run-event-details-llm-output')).not.toHaveText('', { timeout: 120000 });

  await tracePage.close();
}

test.describe('chat trace link', {
  tag: ['@svc_chat_app', '@svc_tracing_app', '@svc_agents_orchestrator', '@svc_gateway', '@svc_organizations'],
}, () => {
  for (const scenario of TRACE_SCENARIOS) {
    test(`view trace opens tracing run (${scenario.name})`, async ({ page }) => {
      test.setTimeout(8 * 60_000);

      const { organizationId, participantId } = await setupTestAgent(page, {
        endpoint: scenario.endpoint,
        protocol: scenario.protocol,
        initImage: scenario.initImage,
      });
      const identityId = await resolveIdentityId(page);
      const chatId = await createChat(page, organizationId, participantId);
      await setSelectedOrganization(page, organizationId);

      const messageText = `trace-${scenario.name}-${randomUUID()}`;
      await sendChatMessage(page, chatId, messageText);

      await waitForAgentReply(page, chatId, identityId, 180_000);

      const messages = await getMessages(page, chatId);
      const userMessage = messages.find(
        (message) => message.body === messageText && message.senderId === identityId,
      );
      if (!userMessage?.id) {
        throw new Error(`Expected to find message id for ${messageText}.`);
      }

      await openTraceFromChat(page, {
        chatId,
        organizationId,
        messageId: userMessage.id,
        messageText,
      });
    });
  }
});
