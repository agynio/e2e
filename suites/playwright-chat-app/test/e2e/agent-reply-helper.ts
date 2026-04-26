import type { Page } from '@playwright/test';
import { expect } from '@playwright/test';
import { sendChatMessage } from './chat-api';
const DEFAULT_TIMEOUT_MS = 45_000;

export async function ensureAgentReply(
  page: Page,
  chatId: string,
  replyText = 'hello',
  timeoutMs = DEFAULT_TIMEOUT_MS,
): Promise<void> {
  const messageList = page.getByTestId('chat-message');

  try {
    await expect.poll(() => messageList.count(), { timeout: timeoutMs }).toBeGreaterThan(1);
    return;
  } catch (error) {
    if (process.env.CI === 'true') {
      throw error;
    }
    await sendChatMessage(page, chatId, replyText);
    await expect.poll(() => messageList.count(), { timeout: timeoutMs }).toBeGreaterThan(1);
  }
}
