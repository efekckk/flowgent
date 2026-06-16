export interface CredentialField {
  key: string;
  label: string;
  type: 'text' | 'password';
  placeholder?: string;
}

export interface CredentialTypeDef {
  value: string;
  label: string;
  fields: CredentialField[];
}

export const CREDENTIAL_TYPES: CredentialTypeDef[] = [
  {
    value: 'openai',
    label: 'OpenAI API key',
    fields: [
      { key: 'api_key', label: 'API key', type: 'password', placeholder: 'sk-...' },
    ],
  },
  {
    value: 'anthropic',
    label: 'Anthropic API key',
    fields: [
      { key: 'api_key', label: 'API key', type: 'password', placeholder: 'sk-ant-...' },
    ],
  },
  {
    value: 'slack_webhook',
    label: 'Slack incoming webhook',
    fields: [
      { key: 'url', label: 'Webhook URL', type: 'password', placeholder: 'https://hooks.slack.com/services/...' },
    ],
  },
  {
    value: 'telegram_bot',
    label: 'Telegram bot token',
    fields: [
      { key: 'bot_token', label: 'Bot token', type: 'password', placeholder: '123456:ABC-...' },
    ],
  },
  {
    value: 'smtp',
    label: 'SMTP credentials',
    fields: [
      { key: 'host',     label: 'Host',         type: 'text',     placeholder: 'smtp.example.com' },
      { key: 'port',     label: 'Port',         type: 'text',     placeholder: '587' },
      { key: 'username', label: 'Username',     type: 'text',     placeholder: 'apikey or username' },
      { key: 'password', label: 'Password',     type: 'password' },
      { key: 'from',     label: 'From address', type: 'text',     placeholder: 'bot@example.com' },
    ],
  },
  {
    value: 'postgres',
    label: 'Postgres DSN',
    fields: [
      { key: 'dsn', label: 'DSN', type: 'password', placeholder: 'postgres://user:pw@host:5432/db?sslmode=require' },
    ],
  },
];

export function fieldsForType(value: string): CredentialField[] {
  return CREDENTIAL_TYPES.find((t) => t.value === value)?.fields ?? [];
}
