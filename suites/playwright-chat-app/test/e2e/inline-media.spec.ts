import { argosScreenshot } from '@argos-ci/playwright';
import type { Page } from '@playwright/test';
import { expect, test } from './fixtures';
import {
  createChat,
  sendChatMessage,
  setupTestAgent,
} from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

const defaultTestLlmEndpoint = 'https://testllm.dev/v1/org/agynio/suite/codex/responses';
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

async function openChat(page: Page, message: string): Promise<void> {
  const { organizationId, participantId } = await setupTestAgent(page, {
    endpoint: llmEndpoint,
    initImage: process.env.E2E_AGENT_INIT_IMAGE,
  });
  const chatId = await createChat(page, organizationId, participantId);
  await setSelectedOrganization(page, organizationId);
  await sendChatMessage(page, chatId, message);
  const chatLoaded = page.waitForResponse(
    (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
    { timeout: 15000 },
  );
  await page.goto(`/chats/${encodeURIComponent(chatId)}`);
  await chatLoaded;
}

function mermaidMessage(source: string) {
  return `\`\`\`mermaid\n${source}\n\`\`\``;
}

function vegaMessage(source: string) {
  return `\`\`\`vega-lite\n${source}\n\`\`\``;
}

test.describe('inline-media', {
  tag: [
    '@svc_chat_app',
    '@svc_gateway',
    '@svc_agents_orchestrator',
    '@svc_organizations',
    '@svc_files',
    '@svc_media_proxy',
  ],
}, () => {
  test('renders mermaid diagrams inline', async ({ page }) => {
    test.setTimeout(180_000);
    await openChat(page, mermaidMessage(mermaidSource));

    const mermaidCanvas = page.getByTestId('markdown-mermaid');
    await expect(mermaidCanvas).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'inline-mermaid');
  });

  test('renders vega-lite charts inline', async ({ page }) => {
    test.setTimeout(180_000);
    await openChat(page, vegaMessage(vegaLiteSource));

    const chart = page.getByTestId('markdown-vega-lite');
    await expect(chart).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'inline-vega-lite');
  });

  test('handles invalid mermaid input', async ({ page }) => {
    test.setTimeout(180_000);
    const invalidMermaidSource = `graph TD
  A -->`;
    await openChat(page, mermaidMessage(invalidMermaidSource));

    await expect(page.getByText('Mermaid render failed')).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'inline-mermaid-invalid');
  });

  test('handles invalid vega-lite input', async ({ page }) => {
    test.setTimeout(180_000);
    const invalidVegaSource = '{"data":{"values":[{"x":1,"y":2}]},"mark":"bar"}';
    await openChat(page, vegaMessage(invalidVegaSource));

    await expect(page.getByTestId('markdown-vega-lite')).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'inline-vega-lite-invalid');
  });

  test('renders multiple inline media attachments', async ({ page }) => {
    test.setTimeout(180_000);
    await openChat(page, `${mermaidMessage(mermaidSource)}\n\n${vegaMessage(vegaLiteSource)}`);

    await expect(page.getByTestId('markdown-mermaid')).toBeVisible({ timeout: 120000 });
    await expect(page.getByTestId('markdown-vega-lite')).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'inline-media-multi');
  });
});
