import { argosScreenshot } from '@argos-ci/playwright';
import { expect, test, type SeededRun } from './fixtures';

const timelinePath = (context: SeededRun) =>
  `/${context.organizationId}/runs/${context.runId}`;

const timelineMask = (page: Parameters<typeof argosScreenshot>[0]) => [
  page.getByTestId('run-event-meta'),
  page.locator('[data-run-event-status]'),
  page.getByTestId('run-summary-status'),
  page.getByTestId('run-summary-duration'),
  page.getByTestId('run-summary-created-at'),
  page.getByTestId('run-summary-tokens'),
];

test.describe('run timeline', { tag: ['@svc_tracing_app', '@smoke'] }, () => {
  test('renders run timeline on load', async ({ page, seededRun }) => {
    await page.goto(timelinePath(seededRun));

    const eventsList = page.getByTestId('run-events-list');
    await expect(eventsList).toBeVisible();
    await expect(page.getByTestId(`run-event-${seededRun.messageEventId}`)).toContainText('Message • Source');
    await expect(page.getByTestId(`run-event-${seededRun.llmEventId}`)).toContainText('LLM Call');
    await argosScreenshot(page, 'run-timeline-loaded', { mask: timelineMask(page) });
  });

  test('shows run summary', async ({ page, seededRun }) => {
    await page.goto(timelinePath(seededRun));

    await expect(page.getByRole('button', { name: /events/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /tokens/ })).toBeVisible();
    await argosScreenshot(page, 'run-timeline-summary', { mask: timelineMask(page) });
  });
});
