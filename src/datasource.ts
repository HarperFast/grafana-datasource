import { DataSourceInstanceSettings, CoreApp, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import {
	HarperQuery,
	HarperDataSourceOptions,
	DEFAULT_QUERY,
	SearchValue,
	ListMetricsResponse,
	DescribeMetricResponse,
	ListMetricsRequest,
	MetricType,
} from './types';

export class DataSource extends DataSourceWithBackend<HarperQuery, HarperDataSourceOptions> {
	constructor(instanceSettings: DataSourceInstanceSettings<HarperDataSourceOptions>) {
		super(instanceSettings);
	}

	getDefaultQuery(_: CoreApp): Partial<HarperQuery> {
		return DEFAULT_QUERY;
	}

	/**
	 *  coerceValue takes a raw form fieldVal string and toType string and attempts to coerce
	 *  fieldVal to toType (or whichever type it looks like if toType === 'auto').
	 *
	 *  toType can be any of 'auto', 'boolean', 'string', 'number', 'number_array'
	 *
	 *  It returns an instance of SearchValue.
	 */
	coerceValue(fieldVal: string, toType: string): SearchValue {
		const trimmedFieldVal = fieldVal.trim();
		const canonicalFieldVal = trimmedFieldVal.toLowerCase();

		if (toType === 'auto' || toType === 'boolean') {
			if (canonicalFieldVal === 'false') {
				return { val: false, type: 'boolean' };
			}

			if (canonicalFieldVal === 'true') {
				return { val: true, type: 'boolean' };
			}
		}

		if (toType === 'auto' || toType === 'number') {
			const numFieldVal = parseFloat(canonicalFieldVal);
			if (!isNaN(numFieldVal) && numFieldVal.toString() === trimmedFieldVal) {
				return { val: numFieldVal, type: 'number' };
			}
		}

		if (toType === 'auto' || toType === 'number_array') {
			const elems = canonicalFieldVal.split(/\s*,\s*/);
			const nums = elems.map((e) => parseFloat(e));
			if (
				nums.every((n, i) => {
					return !isNaN(n) && n.toString() === elems[i];
				})
			) {
				return { val: nums, type: 'number_array' };
			}
		}

		return { val: trimmedFieldVal, type: 'string' };
	}

	applyTemplateVariables(queryTemplate: HarperQuery, scopedVars: ScopedVars) {
		const templateSrv = getTemplateSrv();
		let query = { ...queryTemplate };
		if (queryTemplate.queryAttrs) {
			if ('conditions' in queryTemplate.queryAttrs) {
				const conditions = queryTemplate.queryAttrs?.conditions
					?.filter((c) => c.attribute && c.comparator && c.value?.val)
					.map((c) => {
						const searchFieldVal: any = templateSrv.replace(c.value?.val.toString(), scopedVars);
						const searchValType = c.searchValueType ?? 'auto';
						const searchVal = this.coerceValue(searchFieldVal, searchValType);
						return { attribute: c.attribute, comparator: c.comparator, value: searchVal };
					});
				query.queryAttrs = { ...query.queryAttrs, conditions };
			}
			if ('from' in queryTemplate.queryAttrs || 'to' in queryTemplate.queryAttrs) {
				const from = Number.parseInt(templateSrv.replace(queryTemplate.queryAttrs?.from?.toString(), scopedVars), 10);
				const to = Number.parseInt(templateSrv.replace(queryTemplate.queryAttrs?.to?.toString(), scopedVars), 10);
				query.queryAttrs = { ...query.queryAttrs, from, to };
			}
		}
		return query;
	}

	isSearchByConditionsQuery(query: HarperQuery) {
		return (
			query.operation === 'search_by_conditions' &&
			!!query.queryAttrs &&
			'database' in query.queryAttrs &&
			'table' in query.queryAttrs &&
			'conditions' in query.queryAttrs
		);
	}

	/** This assumes query is a SearchByConditionsQuery, so call isSearchByConditionsQuery first if you're not sure
	 */
	isReadySearchByConditionsQuery(query: HarperQuery) {
		if ('conditions' in query.queryAttrs!) {
			return (
				query.queryAttrs.conditions!.length > 0 &&
				!!query.queryAttrs.conditions![0].attribute &&
				query.queryAttrs.conditions![0].attribute.length > 0 &&
				!!query.queryAttrs.conditions![0].comparator &&
				query.queryAttrs.conditions![0].comparator.length > 0 &&
				!!query.queryAttrs.conditions![0].value?.val
			);
		}
		return false;
	}

	isGetAnalyticsQuery(query: HarperQuery) {
		return query.operation === 'get_analytics' && !!query.queryAttrs && 'metric' in query.queryAttrs;
	}

	isReadyGetAnalyticsQuery(query: HarperQuery) {
		if ('metric' in query.queryAttrs!) {
			return query.queryAttrs.metric!.length > 0;
		}
		return false;
	}

	filterQuery(query: HarperQuery) {
		// prevent the query from being executed until it's minimally valid
		return (
			(this.isSearchByConditionsQuery(query) && this.isReadySearchByConditionsQuery(query)) ||
			(this.isGetAnalyticsQuery(query) && this.isReadyGetAnalyticsQuery(query))
		);
	}

	listMetrics(types?: MetricType[]): Promise<ListMetricsResponse> {
		let params: ListMetricsRequest = { types: [] };
		if (types) {
			params = { types };
		}
		const templateSrv = getTemplateSrv();
		const from = Number(templateSrv.replace('${__from}'));
		if (from && !Number.isNaN(from)) {
			params.customMetricsWindow = Date.now() - from;
		}
		return this.getResource('/metrics', params);
	}

	describeMetric(metric: string): Promise<DescribeMetricResponse> {
		return this.getResource(`/metrics/${metric}`);
	}
}
