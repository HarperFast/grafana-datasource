import { test, expect } from '@grafana/plugin-e2e';
import { HDBDataSourceOptions, HDBSecureJsonData } from '../src/types';

test('"Save & test" should be successful when configuration is valid', async ({
  createDataSourceConfigPage,
  readProvisionedDataSource,
  page,
}) => {
  const ds = await readProvisionedDataSource<HDBDataSourceOptions, HDBSecureJsonData>({ fileName: 'datasources.yml' });
  const configPage = await createDataSourceConfigPage({ type: ds.type });
  await page.getByRole('textbox', { name: 'Operations API URL' }).fill(ds.jsonData.opsAPIURL ?? '');
  await page.getByRole('textbox', { name: 'Username' }).fill(ds.jsonData.username ?? '');
  await page.getByRole('textbox', { name: 'Password' }).fill(ds.secureJsonData?.password ?? '');
  await expect(configPage.saveAndTest()).toBeOK();
});

test('"Save & test" should fail when configuration is invalid', async ({
  createDataSourceConfigPage,
  readProvisionedDataSource,
  page,
}) => {
  const ds = await readProvisionedDataSource<HDBDataSourceOptions, HDBSecureJsonData>({ fileName: 'datasources.yml' });
  const configPage = await createDataSourceConfigPage({ type: ds.type });
  await page.getByRole('textbox', { name: 'Operations API URL' }).fill(ds.jsonData.opsAPIURL ?? '');
  await page.getByRole('textbox', { name: 'Username' }).fill(ds.jsonData.username ?? '');
  await expect(configPage.saveAndTest()).not.toBeOK();
  await expect(configPage).toHaveAlert('error', { hasText: 'no password found for HarperDB connection' });
});
