import { test, expect, PLUGIN_ID } from '../fixtures';

test.describe('Configuration', () => {
  test('renders all configuration sections', async ({ appConfigPage, page }) => {
    await expect(page.getByText('Data Sources')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Per-Environment Datasources')).toBeVisible();
    await expect(page.getByRole('group', { name: 'Authentication' })).toBeVisible();
    await expect(page.getByText('Detection & Overrides')).toBeVisible();
  });

  test('datasource fields are present', async ({ appConfigPage, page }) => {
    await expect(page.getByText('Metrics (Prometheus/Mimir)')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Traces (Tempo)')).toBeVisible();
    await expect(page.getByText('Logs (Loki)')).toBeVisible();
  });

  test('auto-detect and save buttons are actionable', async ({ appConfigPage, page }) => {
    const autoDetect = page.getByRole('button', { name: /Auto-detect/i });
    const save = page.getByRole('button', { name: /Save settings/i });

    await expect(autoDetect).toBeVisible({ timeout: 10_000 });
    await expect(autoDetect).toBeEnabled();
    await expect(save).toBeVisible();
    await expect(save).toBeEnabled();
  });

  // Save calls updatePluginAndReload: success → window.location.reload (no toast);
  // failure → console.error only. Assert the settings POST instead of UI chrome.
  test('save settings posts plugin settings', async ({ appConfigPage, page }) => {
    const save = page.getByRole('button', { name: /Save settings/i });
    await expect(save).toBeVisible({ timeout: 10_000 });

    const responsePromise = page.waitForResponse(
      (r) => r.url().includes(`/api/plugins/${PLUGIN_ID}/settings`) && r.request().method() === 'POST',
      { timeout: 10_000 }
    );
    await save.click();
    const response = await responsePromise;
    // 200 = persisted (page then reloads); 4xx = Grafana rejected the payload.
    // Either proves the submit handler ran.
    expect(response.status(), `unexpected settings status ${response.status()}`).toBeLessThan(500);
  });

  test('authentication section has service account token field', async ({ appConfigPage, page }) => {
    await expect(page.getByText('Grafana Service Account Token')).toBeVisible({ timeout: 10_000 });
  });

  test('detection overrides has namespace and duration fields', async ({ appConfigPage, page }) => {
    await expect(page.getByText('Metric Namespace')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Duration Unit', { exact: true })).toBeVisible();
  });
});
