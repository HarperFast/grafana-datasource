import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { HarperQuery, HarperDataSourceOptions } from './types';

export const plugin = new DataSourcePlugin<DataSource, HarperQuery, HarperDataSourceOptions>(DataSource)
	.setConfigEditor(ConfigEditor)
	.setQueryEditor(QueryEditor);
