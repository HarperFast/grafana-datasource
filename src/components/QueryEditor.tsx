import React, { ChangeEvent } from 'react';
import {
	InlineField,
	Input,
	Stack,
	Alert,
	Checkbox,
	Label,
	Button,
	InlineLabel,
	Combobox,
	ComboboxOption,
	MultiCombobox,
} from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import {
	AnalyticsQueryAttrs,
	Condition,
	HarperDataSourceOptions,
	HarperQuery,
	QueryAttrs,
	SearchByConditionsQueryAttrs,
} from '../types';

type Props = QueryEditorProps<DataSource, HarperQuery, HarperDataSourceOptions>;
type OpQueryProps = {
	operation: string;
	datasource: DataSource;
	query: HarperQuery;
	onQueryAttrsChange: (attrs: QueryAttrs) => void;
};

function toComboboxOption(v: string): ComboboxOption<string> {
	return { label: v, value: v };
}

function toComboboxOptions(vs: string[]): Array<ComboboxOption<string>> {
	return vs.map(toComboboxOption);
}

interface SearchByConditionsQueryProps {
	datasource: DataSource;
	queryAttrs?: SearchByConditionsQueryAttrs;
	onQueryAttrsChange: (attrs: SearchByConditionsQueryAttrs) => void;
}

interface ConditionFormProps extends SearchByConditionsQueryProps {
	datasource: DataSource;
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

function ConditionForm({ datasource, queryAttrs, onQueryAttrsChange, index }: ConditionFormProps) {
	const condition = queryAttrs?.conditions?.[index];

	const searchValueTypes = ['Auto', 'String', 'Number', 'Boolean', 'Number Array'];
	const searchValueTypesOptions = toComboboxOptions(searchValueTypes);

	return (
		<Stack gap={0} direction="row">
			<InlineField label="Search attribute">
				<Input
					id="search-by-conditions-search-attr"
					name="search-attr"
					width={12}
					required
					value={condition?.search_attribute}
					onChange={(e: ChangeEvent<HTMLInputElement>) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						conditions[index] = { ...conditions[index], search_attribute: e.target.value };
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
				/>
			</InlineField>
			<InlineField label="Search type" style={{ marginLeft: '16px' }}>
				<Combobox
					id="search-by-conditions-search-type"
					options={toComboboxOptions(searchTypes)}
					value={condition?.search_type}
					onChange={(v) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						conditions[index] = { ...conditions[index], search_type: v.value };
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
				/>
			</InlineField>
			<InlineField label="Search value" style={{ marginLeft: '16px' }}>
				<Input
					id="search-by-conditions-search-value"
					name="search-value"
					width={12}
					value={condition?.search_value?.val.toString()}
					onChange={(e: ChangeEvent<HTMLInputElement>) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						const svt = condition?.searchValueType ?? 'auto';
						const coercedVal = datasource.coerceValue(e.target.value, svt);
						conditions[index] = {
							...conditions[index],
							search_value: coercedVal,
						};
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
					required
				/>
			</InlineField>
			<InlineField label="Type">
				<Combobox
					id="search-by-conditions-search-value-type"
					options={searchValueTypesOptions}
					value={condition?.searchValueType}
					onChange={(v) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						conditions[index] = {
							...conditions[index],
							searchValueType: v.value,
							search_value: datasource.coerceValue(condition?.search_value?.val.toString() ?? '', v.value ?? 'string'),
						};
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
				/>
			</InlineField>
			{(condition?.searchValueType === 'auto' || condition?.searchValueType !== condition?.search_value?.type) &&
			condition?.search_value ? (
				<InlineLabel width="auto">{condition.search_value.type}</InlineLabel>
			) : null}
			{/*TODO: support nested conditions here*/}
		</Stack>
	);
}

const defaultCondition = { searchValueType: 'auto' };

function newCondition(id: number): Condition {
	return { ...defaultCondition, id: `condition-${id}` };
}

function SearchByConditionsQueryEditor({ datasource, queryAttrs, onQueryAttrsChange }: SearchByConditionsQueryProps) {
	const nextConditionId = React.useRef((queryAttrs?.conditions?.length ?? 0) + 1);

	const onConditionAdd = () => {
		let conditions = [...(queryAttrs?.conditions || [])];
		const condition = newCondition(nextConditionId.current++);
		onQueryAttrsChange({ ...queryAttrs, conditions: [...conditions, condition] });
	};

	const onConditionRemove = (index: number) => {
		let conditions = [...(queryAttrs?.conditions || [])];
		conditions.splice(index, 1);
		onQueryAttrsChange({ ...queryAttrs, conditions });
	};

	// ensure at least one blank condition form shows up
	if (queryAttrs === undefined) {
		onQueryAttrsChange({});
	}
	if (queryAttrs) {
		queryAttrs.conditions ??= [newCondition(0)];
	}
	let conditions = queryAttrs?.conditions;

	return (
		<Stack gap={0} direction="column">
			<InlineField label="Database">
				<Input
					id="search-by-conditions-database"
					name="database"
					value={queryAttrs?.database}
					placeholder="data"
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
				<Combobox
					id="search-by-conditions-operator"
					onChange={(v) => onQueryAttrsChange({ ...queryAttrs, operator: v.value })}
					options={toComboboxOptions(searchOperators)}
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
			{conditions?.map((condition, index) => {
				return (
					<div className="gf-form-inline" key={condition.id}>
						<ConditionForm
							datasource={datasource}
							queryAttrs={queryAttrs}
							onQueryAttrsChange={onQueryAttrsChange}
							index={index}
						/>
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
							+
						</Button>
					</div>
				</div>
			</div>
		</Stack>
	);
}

interface AnalyticsQueryProps {
	datasource: DataSource;
	queryAttrs?: AnalyticsQueryAttrs;
	onQueryAttrsChange: (queryAttrs: AnalyticsQueryAttrs) => void;
}

function AnalyticsQueryEditor({ queryAttrs, onQueryAttrsChange, datasource }: AnalyticsQueryProps) {
	const selectedMetric = queryAttrs?.metric;

	if (queryAttrs?.from === undefined) {
		onQueryAttrsChange({ ...queryAttrs, from: '${__from}' });
	}
	if (queryAttrs?.to === undefined) {
		onQueryAttrsChange({ ...queryAttrs, to: '${__to}' });
	}

	const loadMetrics = React.useCallback(
		async (input: string) => {
			// TODO: Filter metrics based on `input`
			const metrics = await datasource.listMetrics();
			return toComboboxOptions(metrics);
		},
		[datasource]
	);

	const loadAttributes = React.useCallback(
		async (input: string) => {
			if (selectedMetric) {
				// TODO: Filter attrs based on `input`
				const desc = await datasource.describeMetric(selectedMetric);
				return toComboboxOptions(desc.attributes);
			}
			return [];
		},
		[datasource, selectedMetric]
	);

	return (
		<Stack gap={0} direction="column">
			<InlineField label="Metric">
				<Combobox
					id="analytics-metric"
					options={loadMetrics}
					value={queryAttrs?.metric ? toComboboxOption(queryAttrs.metric) : null}
					onChange={(v) => {
						onQueryAttrsChange({ ...queryAttrs, metric: v.value, attributes: [] });
					}}
				/>
			</InlineField>
			<InlineField label="Attributes">
				<MultiCombobox
					id="analytics-attributes"
					width="auto"
					minWidth={25}
					placeholder="*"
					options={loadAttributes}
					value={queryAttrs?.attributes ? toComboboxOptions(queryAttrs.attributes) : []}
					onChange={(vs: Array<ComboboxOption<string>>) => {
						onQueryAttrsChange({ ...queryAttrs, attributes: vs.map((v) => v.value) });
					}}
				/>
			</InlineField>
			<InlineField label="Start time">
				<Input
					id="analytics-start-time"
					name="start-time"
					value={queryAttrs?.from}
					onChange={(e: ChangeEvent<HTMLInputElement>) => onQueryAttrsChange({ ...queryAttrs, from: e.target.value })}
				/>
			</InlineField>
			<InlineField label="End time">
				<Input
					id="analytics-end-time"
					name="end-time"
					value={queryAttrs?.to}
					onChange={(e: ChangeEvent<HTMLInputElement>) => onQueryAttrsChange({ ...queryAttrs, to: e.target.value })}
				/>
			</InlineField>
		</Stack>
	);
}

function OpQueryEditor({ operation, datasource, query, onQueryAttrsChange }: OpQueryProps) {
	switch (operation) {
		case 'get_analytics':
			return (
				<AnalyticsQueryEditor
					datasource={datasource}
					queryAttrs={query.queryAttrs as AnalyticsQueryAttrs}
					onQueryAttrsChange={onQueryAttrsChange}
				/>
			);
		case 'search_by_conditions':
			return (
				<SearchByConditionsQueryEditor
					datasource={datasource}
					queryAttrs={query.queryAttrs as SearchByConditionsQueryAttrs}
					onQueryAttrsChange={onQueryAttrsChange}
				/>
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

export function QueryEditor({ datasource, query, onChange, onRunQuery }: Props) {
	const operations = ['get_analytics', 'search_by_conditions'];

	const onQueryAttrsChange = (attrs: QueryAttrs) => {
		onChange({ ...query, queryAttrs: attrs });
		onRunQuery();
	};

	const onOperationChange = (operation?: string) => {
		onChange({ ...query, operation: operation });
	};

	// set default op to first one
	let { operation } = query;
	if (operation === undefined) {
		onOperationChange(operations[0]);
	}

	const operationOptions = toComboboxOptions(operations);

	return (
		<Stack gap={2} direction="column">
			<InlineField label="Operation">
				<Combobox
					id="query-editor-operation"
					options={operationOptions}
					onChange={({ value }) => onOperationChange(value)}
					value={operation}
					width={40}
				/>
			</InlineField>
			{operation ? (
				<OpQueryEditor
					operation={operation}
					datasource={datasource}
					query={query}
					onQueryAttrsChange={onQueryAttrsChange}
				/>
			) : null}
		</Stack>
	);
}
