import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

type Sort = {
	attribute?: string;
	descending?: boolean;
	next?: Sort;
};

export interface SearchValue {
	val: string | number | number[] | boolean;
	type: string;
}

export type Condition = {
	id?: string;
	attribute?: string;
	comparator?: string;
	value?: SearchValue;
	searchValueType?: string;
	operator?: string;
	conditions?: Condition[];
};

export interface SearchByConditionsQueryAttrs {
	database?: string;
	table?: string;
	operator?: string;
	sort?: Sort;
	get_attributes?: string[];
	conditions?: Condition[];
	attributes?: string[];
}

export interface AnalyticsQueryAttrs {
	metric?: string;
	attributes?: string[];
	from?: string | number;
	to?: string | number;
	conditions?: Condition[];
}

export type QueryAttrs = SearchByConditionsQueryAttrs | AnalyticsQueryAttrs;

export interface HarperQuery extends DataQuery {
	operation?: string;
	queryAttrs?: QueryAttrs;
}

export const DEFAULT_QUERY: Partial<HarperQuery> = {
	operation: 'get_analytics',
};

/**
 * These are options configured for each DataSource instance
 */
export interface HarperDataSourceOptions extends DataSourceJsonData {
	opsAPIURL?: string;
	username?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface HarperSecureJsonData {
	password?: string;
}

export type MetricType = 'builtin' | 'custom';

export interface ListMetricsRequest {
	types: MetricType[];
	customMetricsWindow?: number;
}

export type ListMetricsResponse = string[];

export interface MetricAttribute {
	name: string;
	type: string;
}

export interface DescribeMetricResponse {
	attributes: MetricAttribute[];
}
