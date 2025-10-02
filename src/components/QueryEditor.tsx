import React, { ChangeEvent } from 'react';
import {
	InlineField,
	Input,
	Stack,
	Alert,
	Label,
	Button,
	InlineLabel,
	Combobox,
	ComboboxOption,
	MultiCombobox,
} from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import type {
	AnalyticsQueryAttrs,
	Condition,
	HarperDataSourceOptions,
	HarperQuery,
	MetricAttribute,
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

interface QueryProps {
	datasource: DataSource;
	queryAttrs?: SearchByConditionsQueryAttrs;
	onQueryAttrsChange: (attrs: SearchByConditionsQueryAttrs) => void;
}

interface ConditionFormProps extends QueryProps {
	datasource: DataSource;
	loadAttributes: (input: string) => Promise<Array<ComboboxOption<string>>>;
	index: number;
}

interface ConditionsFormProps extends QueryProps {
	onConditionAdd: () => void;
	onConditionRemove: (index: number) => void;
	loadAttributes: (input: string) => Promise<Array<ComboboxOption<string>>>;
}

const comparators = [
	'equals',
	'not_equal',
	'contains',
	'starts_with',
	'ends_with',
	'greater_than',
	'greater_than_equal',
	'less_than',
	'less_than_equal',
	'between',
];

function ConditionForm({ datasource, queryAttrs, onQueryAttrsChange, loadAttributes, index }: ConditionFormProps) {
	const condition = queryAttrs?.conditions?.[index];

	const searchValueTypes = ['Auto', 'String', 'Number', 'Boolean', 'Number Array'];
	const searchValueTypesOptions = toComboboxOptions(searchValueTypes);

	return (
		<Stack gap={0} direction="row">
			<InlineField label="Attribute">
				<Combobox
					id={`conditions-search-attr-${index}`}
					width="auto"
					minWidth={12}
					options={loadAttributes}
					value={condition?.attribute}
					onChange={(v: ComboboxOption<string>) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						conditions[index] = { ...conditions[index], attribute: v.value };
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
				/>
			</InlineField>
			<InlineField label="Comparator" style={{ marginLeft: '16px' }}>
				<Combobox
					id={`conditions-comparator-${index}`}
					width="auto"
					minWidth={12}
					options={toComboboxOptions(comparators)}
					value={condition?.comparator}
					onChange={(v) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						conditions[index] = { ...conditions[index], comparator: v.value };
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
				/>
			</InlineField>
			<InlineField label="Value" style={{ marginLeft: '16px' }}>
				<Input
					id={`conditions-search-value-${index}`}
					name="search-value"
					width={20}
					value={condition?.value?.val.toString()}
					onChange={(e: ChangeEvent<HTMLInputElement>) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						const svt = condition?.searchValueType ?? 'auto';
						const coercedVal = datasource.coerceValue(e.target.value, svt);
						conditions[index] = {
							...conditions[index],
							value: coercedVal,
						};
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
					required
				/>
			</InlineField>
			<InlineField label="Type">
				<Combobox
					id={`conditions-search-value-type-${index}`}
					width="auto"
					minWidth={12}
					options={searchValueTypesOptions}
					value={condition?.searchValueType}
					onChange={(v) => {
						const conditions = [...(queryAttrs?.conditions || [])];
						conditions[index] = {
							...conditions[index],
							searchValueType: v.value,
							value: datasource.coerceValue(condition?.value?.val.toString() ?? '', v.value ?? 'string'),
						};
						onQueryAttrsChange({ ...queryAttrs, conditions });
					}}
				/>
			</InlineField>
			{(condition?.searchValueType === 'auto' || condition?.searchValueType !== condition?.value?.type) &&
			condition?.value ? (
				<InlineLabel width="auto">{condition.value.type}</InlineLabel>
			) : null}
			{/*TODO: support nested conditions here*/}
		</Stack>
	);
}

function ConditionsForm({
	datasource,
	queryAttrs,
	onQueryAttrsChange,
	loadAttributes,
	onConditionAdd,
	onConditionRemove,
}: ConditionsFormProps) {
	const conditions = queryAttrs?.conditions;
	return (
		<div>
			{conditions?.map((condition, index) => {
				return (
					<div className="gf-form-inline" key={condition.id}>
						<ConditionForm
							datasource={datasource}
							queryAttrs={queryAttrs}
							onQueryAttrsChange={onQueryAttrsChange}
							loadAttributes={loadAttributes}
							index={index}
						/>
						<Button
							className="btn btn-danger btn-small"
							icon="trash-alt"
							aria-label="Remove condition"
							variant="destructive"
							fill="outline"
							size="sm"
							style={{ margin: '5px' }}
							onClick={(e) => {
								onConditionRemove(index);
								e.preventDefault();
							}}
						/>
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
		</div>
	);
}

const defaultCondition = { searchValueType: 'auto' };

function newCondition(id: number): Condition {
	return { ...defaultCondition, id: `condition-${id}` };
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

	const [attributes, setAttributes] = React.useState<MetricAttribute[]>([]);

	const loadAttributes = React.useCallback(
		async (input: string) => {
			if (selectedMetric) {
				// TODO: Filter attrs based on `input`
				const desc = await datasource.describeMetric(selectedMetric);
				setAttributes(desc.attributes);
				return toComboboxOptions(desc.attributes.map((a) => a.name));
			}
			return [];
		},
		[datasource, selectedMetric]
	);

	const loadNumericAttributes = React.useCallback(
		async (input: string) => {
			if (selectedMetric) {
				const desc = await datasource.describeMetric(selectedMetric);
				const attrs = desc.attributes;
				setAttributes(attrs);
				const numericAttrs = attrs.filter((a) => a.type === 'number');
				return toComboboxOptions(numericAttrs.map((a) => a.name));
			}
			return [];
		},
		[selectedMetric, datasource]
	);

	const selectedAttrsValue = React.useMemo(() => {
		return queryAttrs?.attributes
			? toComboboxOptions(
					queryAttrs.attributes.filter((attr) => {
						const numericAttrs = attributes.filter((a) => a.type === 'number');
						return numericAttrs.map((na) => na.name).includes(attr);
					})
			  )
			: [];
	}, [queryAttrs, attributes]);

	const updateSelectedAttrs = React.useCallback(
		(attrs: ComboboxOption[]) => {
			const labelAttrs = attributes.filter((attr) => attr.type !== 'number');
			let selectedAttrs: string[] = [];
			if (attrs.length > 0) {
				// ensure label attrs are always included
				selectedAttrs = [...labelAttrs.map((a) => a.name), ...attrs.map((a) => a.value)];
			}
			onQueryAttrsChange({ ...queryAttrs, attributes: selectedAttrs });
		},
		[queryAttrs, attributes, onQueryAttrsChange]
	);

	const nextConditionId = React.useRef((queryAttrs?.conditions?.length ?? 0) + 1);

	const onConditionAdd = React.useCallback(() => {
		let conditions = [...(queryAttrs?.conditions || [])];
		const condition = newCondition(nextConditionId.current++);
		onQueryAttrsChange({ ...queryAttrs, conditions: [...conditions, condition] });
	}, [queryAttrs, onQueryAttrsChange]);

	const onConditionRemove = React.useCallback(
		(index: number) => {
			let conditions = [...(queryAttrs?.conditions || [])];
			conditions.splice(index, 1);
			onQueryAttrsChange({ ...queryAttrs, conditions });
		},
		[queryAttrs, onQueryAttrsChange]
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
			<InlineField label="Select attributes">
				<MultiCombobox
					id="analytics-attributes"
					width="auto"
					minWidth={25}
					placeholder="*"
					options={loadNumericAttributes}
					value={selectedAttrsValue}
					onChange={updateSelectedAttrs}
				/>
			</InlineField>
			<Label style={{ marginTop: '25px' }}>Conditions</Label>
			<ConditionsForm
				datasource={datasource}
				queryAttrs={queryAttrs}
				loadAttributes={loadAttributes}
				onQueryAttrsChange={onQueryAttrsChange}
				onConditionAdd={onConditionAdd}
				onConditionRemove={onConditionRemove}
			/>
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
	// Supporting search_by_conditions again would require adapting it to the new backend data ingestion scheme
	// get_analytics uses in datasource.go.
	// const operations = ['get_analytics', 'search_by_conditions'];
	const operations = ['get_analytics'];

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
			{operations.length > 1 ? (
				<InlineField label="Operation">
					<Combobox
						id="query-editor-operation"
						options={operationOptions}
						onChange={({ value }) => onOperationChange(value)}
						value={operation}
						width={40}
					/>
				</InlineField>
			) : null}
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
