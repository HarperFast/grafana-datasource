import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { HDBQuery, HDBDataSourceOptions } from './types';

export const plugin = new DataSourcePlugin<DataSource, HDBQuery, HDBDataSourceOptions>(DataSource)
	.setConfigEditor(ConfigEditor)
	.setQueryEditor(QueryEditor);
