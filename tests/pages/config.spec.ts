import { test, expect, expectAnyVisible } from '../fixtures';

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

  test('save settings is clickable and yields a success or validation response', async ({
    appConfigPage,
    page,
  }) => {
    const save = page.getByRole('button', { name: /Save settings/i });
    await expect(save).toBeVisible({ timeout: 10_000 });
    await save.click();

    // CI / local-without-DS often returns a validation or backend error toast;
    // a blank click with neither success nor error means the handler broke.
    await expectAnyVisible(
      [
        page.locator('[data-testid="data-testid Alert success"]'),
        page.locator('[data-testid="data-testid Alert error"]'),
        page.getByRole('alert'),
      ],
      { timeout: 8_000, message: 'Save settings produced no success or error feedback' }
    );
  });

  test('authentication section has service account token field', async ({ appConfigPage, page }) => {
    await expect(page.getByText('Grafana Service Account Token')).toBeVisible({ timeout: 10_000 });
  });

  test('detection overrides has namespace and duration fields', async ({ appConfigPage, page }) => {
    await expect(page.getByText('Metric Namespace')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Duration Unit', { exact: true })).toBeVisible();
  });
});
