import { DataSourceInstanceSettings, CoreApp, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import {
	HarperQuery,
	HarperDataSourceOptions,
	DEFAULT_QUERY,
	SearchValue,
	ListMetricsResponse,
	DescribeMetricResponse
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
				return {val: false, type: 'boolean'};
			}

			if (canonicalFieldVal === 'true') {
				return {val: true, type: 'boolean'};
			}
		}

		if (toType === 'auto' || toType === 'number') {
			const numFieldVal = parseFloat(canonicalFieldVal);
			if (!isNaN(numFieldVal) && numFieldVal.toString() === trimmedFieldVal) {
				return {val: numFieldVal, type: 'number'};
			}
		}

		if (toType === 'auto' || toType === 'number_array') {
			const elems = canonicalFieldVal.split(/\s*,\s*/);
			const nums = elems.map((e) => parseFloat(e));
			if (nums.every((n, i) => {
				return !isNaN(n) && n.toString() === elems[i]
			})) {
				return {val: nums, type: 'number_array'};
			}
		}

		return {val: trimmedFieldVal, type: 'string'};
	};

	applyTemplateVariables(query: HarperQuery, scopedVars: ScopedVars) {
		const templateSrv = getTemplateSrv();
		const conditions = query.queryAttrs?.conditions?.map(c => {
			const searchFieldVal: any = templateSrv.replace(c.search_value?.val.toString(), scopedVars);
			const searchValType = c.searchValueType ?? 'auto';
			const searchVal = this.coerceValue(searchFieldVal, searchValType);
			return { ...c, search_value: searchVal }
		});
		return {
			...query,
			queryAttrs: { ...query.queryAttrs, conditions },
		};
	}

	filterQuery(query: HarperQuery) {
		// if no query has been provided, prevent the query from being executed
		return !!query.queryAttrs &&
			!!query.queryAttrs.database &&
			!!query.queryAttrs.table &&
			!!query.queryAttrs.conditions &&
			query.queryAttrs.conditions.length > 0 &&
			!!query.queryAttrs.conditions[0].search_attribute &&
			query.queryAttrs.conditions[0].search_attribute.length > 0 &&
			!!query.queryAttrs.conditions[0].search_type &&
			query.queryAttrs.conditions[0].search_type.length > 0 &&
			!!query.queryAttrs.conditions[0].search_value?.val;
	}
}
