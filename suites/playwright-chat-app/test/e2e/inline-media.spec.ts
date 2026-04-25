import { argosScreenshot } from '@argos-ci/playwright';
import type { Page } from '@playwright/test';
import { expect, test } from './fixtures';
import {
  createChat,
  resolveIdentityId,
  sendChatMessage,
  setupTestAgent,
  waitForAgentReply,
} from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

const defaultTestLlmEndpoint = 'https://test-llm.agyn.dev';
const llmEndpoint = process.env.E2E_TEST_LLM_ENDPOINT ?? defaultTestLlmEndpoint;

const mermaidSource = `graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Option 1]
    B -->|No| D[Option 2]
    C --> E[End]
    D --> E`;

const vegaLiteSource = JSON.stringify({
  $schema: 'https://vega.github.io/schema/vega-lite/v5.json',
  description: 'A simple bar chart with embedded data.',
  data: {
    values: [
      { category: 'A', amount: 28 },
      { category: 'B', amount: 55 },
      { category: 'C', amount: 43 },
    ],
  },
  mark: 'bar',
  encoding: {
    x: { field: 'category', type: 'nominal' },
    y: { field: 'amount', type: 'quantitative' },
  },
});

async function openChat(page: Page, pageUrl: string, message: string) {
  const { organizationId } = await setupTestAgent(page, {
    endpoint: llmEndpoint,
    initImage: process.env.E2E_AGENT_INIT_IMAGE,
  });
  const identityId = await resolveIdentityId(page);
  const chatId = await createChat(page, organizationId, identityId);
  await setSelectedOrganization(page, organizationId);
  await sendChatMessage(page, chatId, message);
  const chatLoaded = page.waitForResponse(
    (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
    { timeout: 15000 },
  );
  await page.goto(pageUrl);
  await chatLoaded;
  return { chatId, identityId };
}

async function waitForReply(page: Page, chatId: string, identityId: string) {
  await waitForAgentReply(page, chatId, identityId);
  await page.reload();
}

test.describe('inline-media', { tag: ['@svc_chat_app', '@svc_gateway', '@svc_agents_orchestrator'] }, () => {
  test('renders mermaid diagrams inline', async ({ page }) => {
    test.setTimeout(120_000);
    const message = `Please respond with a mermaid diagram only.
${mermaidSource}`;

    const { chatId, identityId } = await openChat(page, '/chats', message);
    await waitForReply(page, chatId, identityId);

    const mermaidCanvas = page.getByTestId('chat-message-attachment-mermaid');
    await expect(mermaidCanvas).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'inline-mermaid');
  });

  test('renders vega-lite charts inline', async ({ page }) => {
    test.setTimeout(120_000);
    const message = `Please respond with a vega-lite chart only.
${vegaLiteSource}`;

    const { chatId, identityId } = await openChat(page, '/chats', message);
    await waitForReply(page, chatId, identityId);

    const chart = page.getByTestId('chat-message-attachment-vega-lite');
    await expect(chart).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'inline-vega-lite');
  });

  test('handles invalid mermaid input', async ({ page }) => {
    test.setTimeout(120_000);
    const message = `Please respond with an invalid mermaid diagram only.
graph TD
  A -->`;

    const { chatId, identityId } = await openChat(page, '/chats', message);
    await waitForReply(page, chatId, identityId);

    await expect(page.getByText('Mermaid diagram failed to render')).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'inline-mermaid-invalid');
  });

  test('handles invalid vega-lite input', async ({ page }) => {
    test.setTimeout(120_000);
    const message = `Please respond with invalid vega-lite json only.
{"data":{"values":[{"x":1,"y":2}]},"mark":"bar"}`;

    const { chatId, identityId } = await openChat(page, '/chats', message);
    await waitForReply(page, chatId, identityId);

    await expect(page.getByText('Failed to render Vega-Lite chart')).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'inline-vega-lite-invalid');
  });

  test('renders multiple inline media attachments', async ({ page }) => {
    test.setTimeout(120_000);
    const message = `Please respond with a mermaid diagram followed by a vega-lite chart.
Mermaid:
${mermaidSource}
Vega-lite:
${vegaLiteSource}`;

    const { chatId, identityId } = await openChat(page, '/chats', message);
    await waitForReply(page, chatId, identityId);

    await expect(page.getByTestId('chat-message-attachment-mermaid')).toBeVisible({ timeout: 120000 });
    await expect(page.getByTestId('chat-message-attachment-vega-lite')).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'inline-media-multi');
  });
});
