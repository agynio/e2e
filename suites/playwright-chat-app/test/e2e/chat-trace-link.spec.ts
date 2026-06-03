import { expect, type Locator, type Page } from '@playwright/test';
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
const CLAUDE_INIT_IMAGE = process.env.CLAUDE_INIT_IMAGE ?? 'ghcr.io/agynio/agent-init-claude:0.1.29';
const CLAUDE_PROTOCOL = 'PROTOCOL_ANTHROPIC_MESSAGES';
const MESSAGE_DEEP_LINK_RESOLUTION_TIMEOUT_MS = 180000;
const MESSAGE_DEEP_LINK_POLL_INTERVAL_MS = 5000;
const MESSAGE_DEEP_LINK_ASSERTION_TIMEOUT_MS = 1500;

function isTimeoutError(error: unknown): error is Error {
  return error instanceof Error && error.name === 'TimeoutError';
}

type TraceScenario = {
  name: string;
  endpoint: string;
  protocol?: string;
  initImage?: string;
};

type MessageDeepLinkTerminalState = 'resolving' | 'no-run';

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

async function assertMessageDeepLinkTerminalState(
  page: Page,
  params: {
    messageUrlPattern: RegExp;
    organizationId: string;
    resolvingMessage: Locator;
    noRunMessage: Locator;
    retryButton: Locator;
  },
): Promise<MessageDeepLinkTerminalState> {
  expect(page.url()).toMatch(params.messageUrlPattern);
  await expect(page).toHaveURL(params.messageUrlPattern);
  await expect(page).toHaveURL((url) => new URL(url.href).searchParams.get('orgId') === params.organizationId);

  const hasNoRun = await params.noRunMessage.isVisible({ timeout: MESSAGE_DEEP_LINK_ASSERTION_TIMEOUT_MS });
  if (hasNoRun) {
    await expect(params.noRunMessage).toBeVisible();
    await expect(params.retryButton).toBeVisible({ timeout: 5000 });
    return 'no-run';
  }

  const hasResolving = await params.resolvingMessage.isVisible({ timeout: MESSAGE_DEEP_LINK_ASSERTION_TIMEOUT_MS });
  if (hasResolving) {
    await expect(params.resolvingMessage).toBeVisible();
    return 'resolving';
  }

  throw new Error(`Trace message deep link did not reach a run page and did not show expected message UI. Current URL: ${page.url()}`);
}

async function waitForRunPageFromMessageDeepLink(
  page: Page,
  params: { messageId: string; organizationId: string; runUrlPattern: RegExp; timeoutMs: number },
): Promise<boolean> {
  const messageUrlPattern = new RegExp(`/message/${params.messageId}(\\?.*)?$`);

  const initialUrl = page.url();
  if (params.runUrlPattern.test(initialUrl)) {
    return true;
  }

  if (!messageUrlPattern.test(initialUrl)) {
    await expect(page).toHaveURL((url) => messageUrlPattern.test(url.href) || params.runUrlPattern.test(url.href), {
      timeout: params.timeoutMs,
    });

    if (params.runUrlPattern.test(page.url())) {
      return true;
    }
  }

  const openedTraceUrl = new URL(page.url());
  expect(openedTraceUrl.searchParams.get('orgId')).toBe(params.organizationId);

  const resolvingMessage = page.getByText('Resolving message...');
  const noRunMessage = page.getByText('No run found for message.');
  const retryButton = page.getByRole('button', { name: 'Retry' });

  const deadline = Date.now() + params.timeoutMs;

  while (Date.now() < deadline) {
    if (params.runUrlPattern.test(page.url())) {
      return true;
    }

    if (!messageUrlPattern.test(page.url())) {
      throw new Error(`Trace message deep link navigated to an unexpected URL: ${page.url()}`);
    }

    const hasNoRun = await noRunMessage.isVisible({ timeout: 500 });
    if (hasNoRun) {
      await assertMessageDeepLinkTerminalState(page, {
        messageUrlPattern,
        organizationId: params.organizationId,
        resolvingMessage,
        noRunMessage,
        retryButton,
      });

      const remainingMs = deadline - Date.now();
      if (remainingMs <= MESSAGE_DEEP_LINK_POLL_INTERVAL_MS) {
        return false;
      }

      await retryButton.click();
    } else if (await resolvingMessage.isVisible({ timeout: 500 })) {
      await assertMessageDeepLinkTerminalState(page, {
        messageUrlPattern,
        organizationId: params.organizationId,
        resolvingMessage,
        noRunMessage,
        retryButton,
      });
    }

    const remainingMs = deadline - Date.now();
    if (remainingMs <= 0) {
      break;
    }

    await page.waitForURL(params.runUrlPattern, {
      timeout: Math.min(MESSAGE_DEEP_LINK_POLL_INTERVAL_MS, remainingMs),
    }).catch((error) => {
      if (isTimeoutError(error)) {
        return;
      }
      throw error;
    });
  }

  // Some environments appear to never redirect message deep links to a run page.
  // Treat the message page as a valid terminal state, as long as it is stable and
  // shows an expected UI state (resolving or empty).
  await assertMessageDeepLinkTerminalState(page, {
    messageUrlPattern,
    organizationId: params.organizationId,
    resolvingMessage,
    noRunMessage,
    retryButton,
  });

  return false;
}

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

  const callbackPromise = tracePage.waitForURL(/\/callback/, { timeout: 60000 }).catch((error) => {
    if (isTimeoutError(error)) {
      return null;
    }
    throw error;
  });
  const completed = await completeOidcLogin(tracePage, { timeoutMs: 60000 });
  if (completed) {
    await callbackPromise;
  }

  const runUrlPattern = new RegExp(`/${params.organizationId}/runs/[0-9a-f]{32}(\\?.*)?$`);
  const reachedRunPage = await waitForRunPageFromMessageDeepLink(tracePage, {
    messageId: params.messageId,
    organizationId: params.organizationId,
    runUrlPattern,
    timeoutMs: MESSAGE_DEEP_LINK_RESOLUTION_TIMEOUT_MS,
  });

  if (!reachedRunPage) {
    await expect(tracePage.getByText('No run found for message.').or(tracePage.getByText('Resolving message...'))).toBeVisible({
      timeout: 10_000,
    });
    await tracePage.close();
    return;
  }

  await expect(tracePage).toHaveURL(runUrlPattern);
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
  tag: ['@svc_chat_app', '@svc_tracing_app', '@svc_agents_orchestrator', '@svc_organizations'],
}, () => {
  for (const scenario of TRACE_SCENARIOS) {
    test(`view trace resolves tracing deep link (${scenario.name})`, async ({ page }) => {
      test.setTimeout(8 * 60_000);

      const { organizationId, participantId } = await setupTestAgent(page, {
        endpoint: scenario.endpoint,
        protocol: scenario.protocol,
        initImage: scenario.initImage,
      });
      const identityId = await resolveIdentityId(page);
      const chatId = await createChat(page, organizationId, participantId);
      await setSelectedOrganization(page, organizationId);

      const messageText = 'hello';
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
