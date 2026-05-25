import { test, expect } from '@playwright/test';
import { AgentAvailability } from '../../src/gen/agynio/api/agents/v1/agents_pb';
import { buildCreateAgentPayload, buildCreateAgentRequestBytes, buildCreateAgentRequestJson } from './gateway-api';

const createAgentOptions = {
  organizationId: 'organization-id',
  name: 'agent-name',
  role: 'assistant',
  model: 'model-id',
  description: 'description',
  configuration: '{}',
  image: 'alpine:3.21',
  initImage: 'ghcr.io/agynio/agent-init-agn:latest',
};

test.describe('tracing gateway api helpers', () => {
  test('CreateAgent payload sets internal availability enum', () => {
    const payload = buildCreateAgentPayload(createAgentOptions);

    expect(JSON.parse(JSON.stringify(payload))).toEqual({
      ...createAgentOptions,
      availability: AgentAvailability.INTERNAL,
    });
  });

  test('CreateAgent ConnectRPC JSON uses protobuf enum name', () => {
    const payload = buildCreateAgentRequestJson(createAgentOptions);

    expect(JSON.parse(JSON.stringify(payload))).toEqual({
      ...createAgentOptions,
      availability: 'AGENT_AVAILABILITY_INTERNAL',
    });
  });

  test('CreateAgent ConnectRPC proto bytes include availability value', () => {
    const payload = buildCreateAgentRequestBytes(createAgentOptions);

    expect(Buffer.from(payload).toString('hex')).toBe(
      '0a0a6167656e742d6e616d651209617373697374616e741a086d6f64656c2d6964220b6465736372697074696f6e2a027b7d320b616c70696e653a332e3231420f6f7267616e697a6174696f6e2d69644a24676863722e696f2f6167796e696f2f6167656e742d696e69742d61676e3a6c61746573746801',
    );
  });
});
