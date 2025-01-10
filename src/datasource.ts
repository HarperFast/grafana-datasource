import { DataSourceInstanceSettings, CoreApp, ScopedVars } from '@grafana/data';
import {DataSourceWithBackend, getTemplateSrv} from '@grafana/runtime';

import { HDBQuery, HDBDataSourceOptions, DEFAULT_QUERY } from './types';

export class DataSource extends DataSourceWithBackend<HDBQuery, HDBDataSourceOptions> {
	constructor(instanceSettings: DataSourceInstanceSettings<HDBDataSourceOptions>) {
		super(instanceSettings);
	}

	getDefaultQuery(_: CoreApp): Partial<HDBQuery> {
		return DEFAULT_QUERY;
	}

	applyTemplateVariables(query: HDBQuery, scopedVars: ScopedVars) {
		const templateSrv = getTemplateSrv();
		const conditions = query.queryAttrs?.conditions?.map(c => {
			let searchVal: any = templateSrv.replace(c.search_value, scopedVars);
			// TODO: This is a quick n' dirty hack for handling numeric search_values.
			// Figure out how to do this right!
			const searchValNum = parseInt(searchVal, 10);
			if (!isNaN(searchValNum)) {
				searchVal = searchValNum;
			}
			return { ...c, search_value: searchVal }
		});
		return {
			...query,
			queryAttrs: { ...query.queryAttrs, conditions },
		};
	}

	filterQuery(query: HDBQuery): boolean {
		// if no query has been provided, prevent the query from being executed
		return !!query.queryAttrs &&
			!!query.queryAttrs.database &&
			!!query.queryAttrs.table &&
			!!query.queryAttrs.conditions &&
			query.queryAttrs.conditions.length > 0;
	}
}
