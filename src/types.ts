import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

type SysInfoQueryAttrs = {
	attributes?: string[]; // TODO: Maybe restrict this to the supported set?
};

type SearchByConditionsQueryAttrs = {
	// TODO
};

export type QueryAttrs = SysInfoQueryAttrs & SearchByConditionsQueryAttrs;

export interface HDBQuery extends DataQuery {
	operation?: string;
	queryAttrs?: QueryAttrs;
}

	operation: 'system_information',
export const DEFAULT_QUERY: Partial<HDBQuery> = {
};

export interface DataPoint {
	Time: number;
	Value: number;
}

export interface DataSourceResponse {
	datapoints: DataPoint[];
}

/**
 * These are options configured for each DataSource instance
 */
export interface HDBDataSourceOptions extends DataSourceJsonData {
	opsAPIURL?: string;
	username?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface HDBSecureJsonData {
	password?: string;
}
