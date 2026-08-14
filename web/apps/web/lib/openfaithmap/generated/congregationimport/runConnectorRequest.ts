export interface IRunConnectorRequest {
    'sourceCode': string;
    /**
     * Connector-specific run parameters (e.g. {"countryCodes": "UY,PY"} for osm). Only connectors implementing ConnectorConfigurable accept a non-empty map here — supplying one for a connector that doesn't returns an error rather than being silently ignored. Omit or leave empty to run with the connector's own deploy-time configuration.
     *
     */
    'parameters'?: { [key: string]: string } | null;
}
