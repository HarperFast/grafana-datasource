import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput } from '@grafana/ui';
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

	return (
		<>
			<InlineField label="Operations API URL" labelWidth={20} interactive tooltip={'URL for Harper operations API'}>
				<Input
					required
					id="config-editor-ops-api-url"
					onChange={onOpsApiUrlChange}
					value={jsonData.opsAPIURL}
					placeholder="Enter the operations API URL for the Harper server, e.g. http://localhost:9925/"
					width={80}
				/>
			</InlineField>
			<InlineField label="Username" labelWidth={20} interactive tooltip={'Harper username'}>
				<Input
					required
					id="config-editor-username"
					onChange={onUsernameChange}
					value={jsonData.username}
					placeholder="Enter your Harper username"
					width={40}
				/>
			</InlineField>
			<InlineField label="Password" labelWidth={20} interactive tooltip={'Harper password'}>
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
			</InlineField>
		</>
	);
}
