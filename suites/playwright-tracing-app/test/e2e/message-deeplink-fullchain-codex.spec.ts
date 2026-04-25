import { test, expect } from './fixtures';
import { createFullChainRun } from './tracing-run';

test.describe('message deep link full chain codex', { tag: ['@svc_tracing_app', '@svc_agents_orchestrator'] }, () => {
  test('resolves message to run and renders timeline', async ({ page }) => {
    test.setTimeout(8 * 60_000);
    const run = await createFullChainRun(page, { sdk: 'codex' });

    await page.goto(`/message/${run.messageId}?orgId=${run.organizationId}`);

    const runUrlPattern = new RegExp(`/${run.organizationId}/runs/[0-9a-f]{32}(\\?.*)?$`);
    await expect(page).toHaveURL(runUrlPattern, { timeout: 60000 });

    const currentUrl = new URL(page.url());
    const runIdFromUrl = currentUrl.pathname.split('/').pop();
    if (!runIdFromUrl || !/^[0-9a-f]{32}$/i.test(runIdFromUrl)) {
      throw new Error(`Run redirect URL did not include a run id: ${page.url()}`);
    }
    expect(runIdFromUrl).toBe(run.runId);

    await expect(page.getByTestId('run-summary-status')).toContainText(/finished/i, { timeout: 120000 });

    const eventsList = page.getByTestId('run-events-list');
    await expect(eventsList).toBeVisible();
    const eventItems = eventsList.locator('[data-testid^="run-event-"]');
    await expect.poll(() => eventItems.count(), { timeout: 120000 }).toBeGreaterThanOrEqual(5);

    await expect(eventsList.getByRole('button', { name: /create_entities/ })).toBeVisible();
    await expect(eventsList.getByRole('button', { name: /list_directory/ })).toBeVisible();

    const messageEvent = eventsList.getByRole('button', { name: /Message • Source/ }).first();
    await messageEvent.click();
    await expect(page.getByTestId('run-event-details-message-content')).toContainText(run.prompt);

    const llmEvents = eventsList.getByRole('button', { name: /LLM Call/ });
    const llmCount = await llmEvents.count();
    if (llmCount === 0) {
      throw new Error('Expected at least one LLM call event in the timeline.');
    }
    await llmEvents.nth(llmCount - 1).click();
    await expect(page.getByTestId('run-event-details-llm-output')).toContainText(run.expectedResponse);
  });
});
