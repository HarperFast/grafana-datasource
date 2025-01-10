import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

type SysInfoQueryAttrs = {
	attributes?: string[]; // TODO: Maybe restrict this to the supported set?
};

type Sort = {
	attribute?: string;
	descending?: boolean;
	next?: Sort;
}

export type Condition = {
	search_attribute?: string;
	search_type?: string;
	search_value?: string;
	operator?: string;
	conditions?: Condition[];
}

type SearchByConditionsQueryAttrs = {
	database?: string;
	table?: string;
	operator?: string;
	sort?: Sort;
	get_attributes?: string[];
	conditions?: Condition[];
}

export type QueryAttrs = SysInfoQueryAttrs & SearchByConditionsQueryAttrs;

export interface HDBQuery extends DataQuery {
	operation?: string;
	queryAttrs?: QueryAttrs;
}

export const DEFAULT_QUERY: Partial<HDBQuery> = {
	operation: 'search_by_conditions',
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
