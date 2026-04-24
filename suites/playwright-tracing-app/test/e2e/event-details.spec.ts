import { argosScreenshot } from '@argos-ci/playwright';
import { expect, formatSnippet, test, timelineForEvent } from './fixtures';

const baseDetailsMask = (page: Parameters<typeof argosScreenshot>[0]) => [
  page.getByTestId('run-event-meta'),
  page.getByTestId('run-summary-status'),
  page.getByTestId('run-summary-duration'),
  page.getByTestId('run-summary-created-at'),
  page.getByTestId('run-summary-tokens'),
  page.getByTestId('run-event-details-meta'),
  page.locator(
    '[data-testid="run-event-details-message-content"], [data-testid="run-event-details-llm-context"], [data-testid="run-event-details-llm-output"]',
  ),
];

const llmDetailsMask = (page: Parameters<typeof argosScreenshot>[0]) => [
  ...baseDetailsMask(page),
  page.locator(
    '[data-testid="run-event-details-provider"], [data-testid="run-event-details-model"], [data-testid="run-event-details-reasoning"], [data-testid="run-event-details-tools"], [data-testid="assistant-context-panel"]',
  ),
];

test.describe('event details', { tag: ['@svc_tracing_app'] }, () => {
  test('shows LLM call details', async ({ page, seededRun }) => {
    await page.goto(timelineForEvent(seededRun, seededRun.llmEventId));

    await expect(page.getByTestId('run-event-details-heading')).toHaveText('LLM Call');
    const context = page.getByTestId('run-event-details-llm-context');
    await expect(context).toBeVisible();

    const messageSnippet = formatSnippet(seededRun.messageText) ?? seededRun.messageText;
    await expect(context).toContainText(messageSnippet);

    const output = page.getByTestId('run-event-details-llm-output');
    await expect(output).toBeVisible();
    const outputSnippet = formatSnippet(seededRun.llmResponseText) ?? seededRun.llmResponseText;
    await expect(output).toContainText(outputSnippet);

    await argosScreenshot(page, 'event-details-llm-call', { mask: llmDetailsMask(page) });
  });

  test('shows invocation message', async ({ page, seededRun }) => {
    await page.goto(timelineForEvent(seededRun, seededRun.messageEventId));

    await expect(page.getByTestId('run-event-details-heading')).toContainText('Message • Source');
    const content = page.getByTestId('run-event-details-message-content');
    await expect(content).toContainText(seededRun.messageText);

    await argosScreenshot(page, 'event-details-invocation-message', { mask: baseDetailsMask(page) });
  });
});
