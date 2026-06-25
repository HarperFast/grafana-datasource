import React, { ChangeEvent } from 'react';
import { css } from '@emotion/css';
import { Field, Divider, Input, SecretInput, Switch, FieldSet, useTheme2 } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { HarperDataSourceOptions, HarperSecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<HarperDataSourceOptions, HarperSecureJsonData> {}

// Local replacement for @grafana/plugin-ui's DataSourceDescription, which is an
// internal-only package not permitted for plugins published to the catalog.
function DataSourceDescription({
	dataSourceName,
	docsLink,
	hasRequiredFields = true,
}: {
	dataSourceName: string;
	docsLink: string;
	hasRequiredFields?: boolean;
}) {
	const theme = useTheme2();
	const styles = {
		container: css({ p: { margin: 0 }, 'p + p': { marginTop: theme.spacing(2) } }),
		text: css({
			...theme.typography.body,
			color: theme.colors.text.secondary,
			a: {
				color: theme.colors.text.link,
				textDecoration: 'underline',
				'&:hover': { textDecoration: 'none' },
			},
		}),
	};
	return (
		<div className={styles.container}>
			<p className={styles.text}>
				Before you can use the {dataSourceName} data source, you must configure it below. For detailed
				instructions,{' '}
				<a href={docsLink} target="_blank" rel="noreferrer">
					view the documentation
				</a>
				.
			</p>
			{hasRequiredFields && (
				<p className={styles.text}>
					<i>Fields marked with * are required</i>
				</p>
			)}
		</div>
	);
}

export function ConfigEditor(props: Props) {
	const { onOptionsChange, options } = props;
	const { jsonData, secureJsonFields, secureJsonData } = options;

	const onOpsApiUrlChange = (event: ChangeEvent<HTMLInputElement>) => {
		onOptionsChange({
			...options,
			jsonData: {
				...jsonData,
				opsAPIURL: event.target.value,
			},
		});
	};

	const onUsernameChange = (event: ChangeEvent<HTMLInputElement>) => {
		onOptionsChange({
			...options,
			jsonData: {
				...jsonData,
				username: event.target.value,
			},
		});
	};

	// Secure field (only sent to the backend)
	const onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
		onOptionsChange({
			...options,
			secureJsonData: {
				password: event.target.value,
			},
		});
	};

	const onResetPassword = () => {
		onOptionsChange({
			...options,
			secureJsonFields: {
				...options.secureJsonFields,
				password: false,
			},
			secureJsonData: {
				...options.secureJsonData,
				password: '',
			},
		});
	};

	const onTlsSkipVerifyChange = (event: ChangeEvent<HTMLInputElement>) => {
		const newValue = !jsonData.tlsSkipVerify;
		onOptionsChange({
			...options,
			jsonData: {
				...jsonData,
				tlsSkipVerify: newValue,
			},
		});
	};

	const onAggregateClusterAnalyticsChange = () => {
		onOptionsChange({
			...options,
			jsonData: {
				...jsonData,
				aggregateClusterAnalytics: !jsonData.aggregateClusterAnalytics,
			},
		});
	};

	return (
		<>
			<DataSourceDescription
				dataSourceName="Harper"
				docsLink="https://github.com/HarperFast/grafana-datasource/blob/main/README.md"
				hasRequiredFields={true}
			/>

			<Divider />

			<FieldSet label="Connection">
				<Field label="Operations API URL" required>
					<Input
						id="config-editor-ops-api-url"
						onChange={onOpsApiUrlChange}
						value={jsonData.opsAPIURL}
						placeholder="Enter the operations API URL for the Harper server, e.g. http://localhost:9925/"
						width={80}
					/>
				</Field>

				<Field
					label="Aggregate analytics across cluster nodes"
					description="Return cluster-wide analytics from this single node by fanning the get_analytics query out to its peers and merging the results (each row is tagged with its origin node). Requires Harper 5.1.13 or later. Leave off to return only this node's analytics, relying on Harper's own analytics replication for cluster-wide data."
				>
					<Switch
						value={!!jsonData.aggregateClusterAnalytics}
						onChange={onAggregateClusterAnalyticsChange}
					/>
				</Field>
			</FieldSet>

			<Divider />

			<FieldSet label="Authentication">
				<Field label="Username" required>
					<Input
						required
						id="config-editor-username"
						onChange={onUsernameChange}
						value={jsonData.username}
						placeholder="Enter your Harper username"
						width={40}
					/>
				</Field>

				<Field label="Password" required>
					<SecretInput
						required
						id="config-editor-password"
						isConfigured={secureJsonFields.password}
						value={secureJsonData?.password}
						placeholder="Enter your Harper password"
						width={40}
						onReset={onResetPassword}
						onChange={onPasswordChange}
					/>
				</Field>

				<Field
					label="Skip TLS Verification"
					description="Don't verify the TLS certificate of the Harper server. Use this option if you're using a self-signed certificate, but only in non-production environments as it is insecure."
				>
					<Switch
						value={jsonData.tlsSkipVerify}
						onChange={onTlsSkipVerifyChange}
					/>
				</Field>
			</FieldSet>
		</>
	);
}
