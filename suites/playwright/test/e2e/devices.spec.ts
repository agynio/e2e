import { promises as fs } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';
import { argosScreenshot } from '@argos-ci/playwright';
import type { Page, Response } from '@playwright/test';
import { test, expect } from './fixtures';
import { createDevice, deleteDevice, listDevices } from './console-api';

const DEVICE_NAME_PREFIX = 'e2e-device';
const CREATE_DEVICE_PATH = '/agynio.api.gateway.v1.UsersGateway/CreateDevice';
const CREATE_DEVICE_RETRIES = 3;
const CREATE_DEVICE_BACKOFF_MS = 500;
const DEVICES_LOCK_PATH = path.join(tmpdir(), 'console-app-devices.lock');
const DEVICES_LOCK_TIMEOUT_MS = 30_000;
const DEVICES_LOCK_RETRY_MS = 200;

const buildDeviceName = (suffix: string) => `${DEVICE_NAME_PREFIX}-${suffix}-${Date.now()}`;

function isCreateDeviceResponse(response: Response, deviceName: string): boolean {
  if (!response.url().includes(CREATE_DEVICE_PATH)) return false;
  const request = response.request();
  if (request.method() !== 'POST') return false;
  try {
    const payload = request.postDataJSON() as { name?: string };
    return payload?.name === deviceName;
  } catch {
    const postData = request.postData();
    return Boolean(postData && postData.includes(deviceName));
  }
}

async function submitCreateDevice(page: Page, deviceName: string): Promise<Response | null> {
  const createDialog = page.getByTestId('devices-create-dialog');
  await createDialog.getByTestId('devices-name').fill(deviceName);
  try {
    const [response] = await Promise.all([
      page.waitForResponse((resp) => isCreateDeviceResponse(resp, deviceName), { timeout: 90_000 }),
      page.getByTestId('devices-submit').click(),
    ]);
    return response;
  } catch (error) {
    if (error instanceof Error && error.name === 'TimeoutError') {
      return null;
    }
    throw error;
  }
}

async function cleanupDevices(page: Page): Promise<void> {
  const devices = await listDevices(page);
  const deviceIds = devices
    .filter((device) => device.name?.startsWith(DEVICE_NAME_PREFIX))
    .map((device) => device.meta?.id)
    .filter((deviceId): deviceId is string => Boolean(deviceId));
  await Promise.all(deviceIds.map((deviceId) => deleteDevice(page, deviceId)));
}

async function acquireDevicesLock(): Promise<() => Promise<void>> {
  const start = Date.now();
  while (true) {
    try {
      const handle = await fs.open(DEVICES_LOCK_PATH, 'wx');
      return async () => {
        await handle.close();
        await fs.unlink(DEVICES_LOCK_PATH).catch(() => undefined);
      };
    } catch (error) {
      if (error instanceof Error && 'code' in error && error.code === 'EEXIST') {
        if (Date.now() - start > DEVICES_LOCK_TIMEOUT_MS) {
          throw new Error('Timed out waiting for devices test lock.');
        }
        await delay(DEVICES_LOCK_RETRY_MS);
        continue;
      }
      throw error;
    }
  }
}

test.describe.serial('devices', { tag: ['@svc_console'] }, () => {
  let releaseDevicesLock: (() => Promise<void>) | null = null;

  test.beforeEach(async ({ page }) => {
    releaseDevicesLock = await acquireDevicesLock();
    try {
      await cleanupDevices(page);
    } catch (error) {
      await releaseDevicesLock();
      releaseDevicesLock = null;
      throw error;
    }
  });

  test.afterEach(async ({ page }) => {
    try {
      await cleanupDevices(page);
    } finally {
      if (releaseDevicesLock) {
        await releaseDevicesLock();
        releaseDevicesLock = null;
      }
    }
  });

  test('shows empty devices page', async ({ page }) => {
    await page.goto('/devices');
    await expect(page.getByTestId('list-search')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('devices-empty')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'devices-empty');
  });

  test('creates a device and shows enrollment JWT', async ({ page }) => {
    test.setTimeout(120_000);
    const deviceName = buildDeviceName('jwt');

    await page.goto('/devices');
    await expect(page.getByTestId('list-search')).toBeVisible({ timeout: 15000 });

    await page.getByTestId('devices-create').click();
    await expect(page.getByTestId('devices-create-dialog')).toBeVisible({ timeout: 15000 });

    const createDialog = page.getByTestId('devices-create-dialog');
    const createButton = page.getByTestId('devices-create');
    const attemptNames = Array.from({ length: CREATE_DEVICE_RETRIES }, (_, index) =>
      index === 0 ? deviceName : `${deviceName}-retry-${index}`,
    );

    let createdName = '';
    let lastStatus: number | null = null;

    for (const attemptName of attemptNames) {
      if (!(await createDialog.isVisible().catch(() => false))) {
        await createButton.click();
        await expect(createDialog).toBeVisible({ timeout: 15000 });
      }

      const response = await submitCreateDevice(page, attemptName);
      lastStatus = response?.status() ?? null;
      if (response?.status() === 200) {
        createdName = attemptName;
        break;
      }
      await page.waitForTimeout(CREATE_DEVICE_BACKOFF_MS);
    }

    if (!createdName) {
      throw new Error(`CreateDevice failed${lastStatus !== null ? ` with status ${lastStatus}` : ''}.`);
    }

    await expect(page.getByTestId('devices-jwt-value')).toBeVisible({ timeout: 30_000 });
    await argosScreenshot(page, 'devices-create-jwt');

    await page.getByTestId('devices-jwt-done').click();

    const deviceRow = page.getByTestId('devices-row').filter({ hasText: createdName });
    await expect(deviceRow).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'devices-list');
  });

  test('deletes a device with confirmation', async ({ page }) => {
    const deviceName = buildDeviceName('delete');
    const created = await createDevice(page, { name: deviceName });
    const deviceId = created.device?.meta?.id;
    if (!deviceId) {
      throw new Error('CreateDevice response missing device id for delete test.');
    }

    await page.goto('/devices');
    await expect(page.getByTestId('list-search')).toBeVisible({ timeout: 15000 });

    const deviceRow = page.getByTestId('devices-row').filter({ hasText: deviceName });
    await expect(deviceRow).toBeVisible({ timeout: 15000 });

    await deviceRow.getByTestId('devices-delete').click();
    await expect(page.getByTestId('confirm-dialog')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'devices-delete-confirm');

    await page.getByTestId('confirm-dialog-confirm').click();
    await expect(deviceRow).toHaveCount(0, { timeout: 15000 });
  });
});
