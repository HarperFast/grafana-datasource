import React, { ChangeEvent } from 'react';
import { InlineField, Input, Stack, Select, Alert, MultiSelect, Checkbox, Label, Button } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from '../datasource';
import { HDBDataSourceOptions, HDBQuery, QueryAttrs, Condition } from '../types';

type Props = QueryEditorProps<DataSource, HDBQuery, HDBDataSourceOptions>;
type OpQueryProps = {
	operation: string;
	query: HDBQuery;
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

interface SearchByConditionsQueryProps {
	queryAttrs?: QueryAttrs;
	onQueryAttrsChange: (attrs: QueryAttrs) => void;
}

interface ConditionFormProps extends SearchByConditionsQueryProps {
	index: number;
}

const searchOperators = ['and', 'or'];
const searchTypes = [
	'equals',
	'contains',
	'starts_with',
	'ends_with',
	'greater_than',
	'greater_than_equal',
	'less_than',
	'less_than_equal',
	'between',
];

function ConditionForm({ queryAttrs, onQueryAttrsChange, index }: ConditionFormProps) {
	const condition = queryAttrs?.conditions?.[index];
	return (
		<Stack gap={0} direction="row">
			<InlineField label="Search attribute">
				<Input
					id="search-by-conditions-search-attr"
					name="search-attr"
					required
					value={condition?.search_attribute}
					onBlur={(e: ChangeEvent<HTMLInputElement>) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						conditions[index] = { ...conditions[index], search_attribute: e.target.value };
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
				/>
			</InlineField>
			<InlineField label="Search type">
				<Select
					id="search-by-conditions-search-type"
					name="search-type"
					options={toSelectableValues(searchTypes)}
					value={condition?.search_type}
					onChange={(v) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						conditions[index] = { ...conditions[index], search_type: v.value };
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
					required
				/>
			</InlineField>
			<InlineField label="Search value">
				<Input
					id="search-by-conditions-search-value"
					name="search-value"
					value={condition?.search_value}
					onBlur={(e: ChangeEvent<HTMLInputElement>) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						conditions[index] = { ...conditions[index], search_value: e.target.value };
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
					required
				/>
			</InlineField>
			{/*TODO: Figure out how to support nested conditions here*/}
		</Stack>
	);
}

function SearchByConditionsQueryEditor({ queryAttrs, onQueryAttrsChange }: SearchByConditionsQueryProps) {
	const onConditionAdd = () => {
		let conditions = [...(queryAttrs?.conditions || [])];
		const defaultCondition: Condition = {};
		onQueryAttrsChange({ ...queryAttrs, conditions: [...conditions, defaultCondition] });
	};

	const onConditionRemove = (index: number) => {
		let conditions = [...(queryAttrs?.conditions || [])];
		conditions.splice(index, 1);
		onQueryAttrsChange({ ...queryAttrs, conditions });
	};

	let conditions = queryAttrs?.conditions;
	// ensure at least one blank condition form shows up
	conditions ||= [{}];

	return (
		<Stack gap={0} direction="column">
			<InlineField label="Database">
				<Input
					id="search-by-conditions-database"
					name="database"
					value={queryAttrs?.database}
					placeholder="data"
					defaultValue="data"
					onChange={(e: ChangeEvent<HTMLInputElement>) =>
						onQueryAttrsChange({ ...queryAttrs, database: e.target.value })
					}
				/>
			</InlineField>
			<InlineField label="Table">
				<Input
					id="search-by-conditions-table"
					name="table"
					onChange={(e: ChangeEvent<HTMLInputElement>) => onQueryAttrsChange({ ...queryAttrs, table: e.target.value })}
					value={queryAttrs?.table}
					required
				/>
			</InlineField>
			<InlineField label="Operator">
				<Select
					id="search-by-conditions-operator"
					onChange={(v) => onQueryAttrsChange({ ...queryAttrs, operator: v.value })}
					options={toSelectableValues(searchOperators)}
					value={queryAttrs?.operator}
					placeholder="and"
				/>
			</InlineField>
			<Stack gap={0}>
				<InlineField label="Sort attribute">
					<Input
						id="search-by-conditions-sort-attr"
						name="sort-attr"
						value={queryAttrs?.sort?.attribute}
						onChange={(e: ChangeEvent<HTMLInputElement>) =>
							onQueryAttrsChange({ ...queryAttrs, sort: { ...queryAttrs?.sort, attribute: e.target.value } })
						}
					/>
				</InlineField>
				<InlineField label="Sort descending?">
					<Checkbox
						id="search-by-conditions-sort-descending"
						name="sort-descending"
						onChange={(e: ChangeEvent<HTMLInputElement>) =>
							onQueryAttrsChange({ ...queryAttrs, sort: { ...queryAttrs?.sort, descending: e.target.checked } })
						}
					/>
				</InlineField>
				{/*Figure out how to support sort.next here*/}
			</Stack>
			<InlineField label="Get attributes" tooltip="Separate multiple attributes with commas (no spaces between)">
				<Input
					id="search-by-conditions-get-attributes"
					name="get-attributes"
					defaultValue="*"
					value={queryAttrs?.attributes}
					onChange={(e: ChangeEvent<HTMLInputElement>) =>
						onQueryAttrsChange({ ...queryAttrs, attributes: e.target.value.split(',') })
					}
				/>
			</InlineField>
			<Label style={{ marginTop: '25px' }}>Conditions</Label>
			{conditions?.map((condition: Condition, index: number) => {
				return (
					<div className="gf-form-inline" key={JSON.stringify(condition) + index}>
						<ConditionForm queryAttrs={queryAttrs} onQueryAttrsChange={onQueryAttrsChange} index={index} />
						{index > 0 ? (
							<Button
								className="btn btn-danger btn-small"
								icon="trash-alt"
								variant="destructive"
								fill="outline"
								size="sm"
								style={{ margin: '5px' }}
								onClick={(e) => {
									onConditionRemove(index);
									e.preventDefault();
								}}
							/>
						) : null}
					</div>
				);
			})}

			<div className="gf-form-inline">
				<div className="gf-form">
					<div className="gf-form gf-form--grow">
						<Button
							variant="secondary"
							size="sm"
							style={{ marginTop: '5px', marginLeft: '5px' }}
							onClick={(e) => {
								onConditionAdd();
								e.preventDefault();
							}}
						>
							Add Condition
						</Button>
					</div>
				</div>
			</div>
		</Stack>
	);
}

function OpQueryEditor({ operation, query, onQueryAttrsChange }: OpQueryProps) {
	switch (operation) {
		// system_information isn't currently used, but leaving here as an example of handling other ops
		case 'system_information':
			return <SysInfoQueryEditor queryAttrs={query.queryAttrs} onQueryAttrsChange={onQueryAttrsChange} />;
		case 'search_by_conditions':
			return <SearchByConditionsQueryEditor queryAttrs={query.queryAttrs} onQueryAttrsChange={onQueryAttrsChange} />;
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
	// const operations = ['search_by_conditions', 'system_information'];
	const operations = ['search_by_conditions'];

	const onQueryAttrsChange = (attrs: QueryAttrs) => {
		onChange({ ...query, queryAttrs: attrs });
		onRunQuery();
	};

	const onOperationChange = (operation?: string) => {
		onChange({ ...query, operation: operation });
	};

	// set default op since we only have one; probably remove this when / if we add others?
	let { operation } = query;
	if (operation === undefined) {
		onOperationChange(operations[0]);
		operation = operations[0];
	}

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
