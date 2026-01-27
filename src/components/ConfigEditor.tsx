import React, { ChangeEvent } from 'react';
import { Field, Divider, Input, SecretInput, Switch } from '@grafana/ui';
import { ConfigSection, DataSourceDescription } from '@grafana/plugin-ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { HarperDataSourceOptions, HarperSecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<HarperDataSourceOptions, HarperSecureJsonData> {}

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

	return (
		<>
			<DataSourceDescription
				dataSourceName="Harper"
				docsLink="https://github.com/HarperFast/grafana-datasource/blob/main/README.md"
				hasRequiredFields={true}
			/>

			<Divider />

			<ConfigSection title="Connection">
				<Field label="Operations API URL" required>
					<Input
						id="config-editor-ops-api-url"
						onChange={onOpsApiUrlChange}
						value={jsonData.opsAPIURL}
						placeholder="Enter the operations API URL for the Harper server, e.g. http://localhost:9925/"
						width={80}
					/>
				</Field>
			</ConfigSection>

			<Divider />

			<ConfigSection title="Authentication">
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
			</ConfigSection>
		</>
	);
}
