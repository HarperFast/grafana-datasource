import React from 'react';
import { InlineField, Input, Stack, Select, Alert, MultiSelect } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from '../datasource';
import { MyDataSourceOptions, MyQuery, QueryAttrs } from '../types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;
type OpQueryProps = {
	operation: string;
	query: MyQuery;
	onQueryAttrsChange: (attrs: QueryAttrs) => void;
};

// TODO: Consider pulling these from a backend resource handler instead of duplicating here
const sysInfoAttrs = [
	'system',
	'time',
	'cpu',
	'memory',
	'disk',
	'network',
	'harperdb_processes',
	'table_size',
	'metrics',
	'threads',
	'replication',
];

function toSelectableValues(vs: string[]): Array<SelectableValue<string>> {
	return vs.map((v) => ({
		label: v,
		value: v,
	}));
}

const sysInfoOptions = toSelectableValues(sysInfoAttrs);

type SysInfoQueryProps = {
	queryAttrs?: QueryAttrs;
	onQueryAttrsChange: (attrs: QueryAttrs) => void;
};

function SysInfoQueryEditor({ queryAttrs, onQueryAttrsChange }: SysInfoQueryProps) {
	return (
		<Stack gap={0}>
			<InlineField label="System Information Attributes" tooltip="Leave empty for 'all'">
				<MultiSelect
					id="query-editor-sys-info"
					options={sysInfoOptions}
					value={queryAttrs?.attributes}
					onChange={(v) => onQueryAttrsChange({ attributes: v.map((sv) => sv.value ?? '') })}
					width={80}
				/>
			</InlineField>
		</Stack>
	);
}

function OpQueryEditor({ operation, query, onQueryAttrsChange }: OpQueryProps) {
	switch (operation) {
		case 'system_information':
			return <SysInfoQueryEditor queryAttrs={query.queryAttrs} onQueryAttrsChange={onQueryAttrsChange} />;
		case 'search_by_conditions':
			return (
				<Stack>
					<InlineField label="Search by Conditions Query">
						<Input id="query-editor-search-conditions" placeholder="Search by Conditions Query" />
					</InlineField>
				</Stack>
			);
		default:
			return (
				<Stack>
					<Alert severity="error" title="Invalid operation">
						Operation {operation} is not currently supported.
					</Alert>
				</Stack>
			);
	}
}

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
	const onQueryAttrsChange = (attrs: QueryAttrs) => {
		onChange({ ...query, queryAttrs: attrs });
	};

	const onOperationChange = (operation?: string) => {
		onChange({ ...query, operation: operation });

		// TODO: Figure out where this goes instead
		// executes the query
		onRunQuery();
	};

	const { operation } = query;

	const operations = ['search_by_conditions', 'system_information'];

	const operationOptions = toSelectableValues(operations);

	return (
		<Stack gap={2} direction="column">
			<InlineField label="Operation">
				<Select
					id="query-editor-operation"
					options={operationOptions}
					onChange={({ value }) => onOperationChange(value)}
					value={operation}
					width={40}
				/>
			</InlineField>
			{operation ? <OpQueryEditor operation={operation} query={query} onQueryAttrsChange={onQueryAttrsChange} /> : null}
		</Stack>
	);
}
