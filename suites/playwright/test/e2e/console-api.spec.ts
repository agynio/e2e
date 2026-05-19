import { test, expect } from '@playwright/test';
import { AgentAvailability } from '../../src/gen/agynio/api/agents/v1/agents_pb';
import { buildCreateAgentPayload, buildCreateAgentRequestBytes, buildCreateAgentRequestJson } from './console-api';

test.describe('console api helpers', () => {
  test('CreateAgent payload defaults availability to internal', () => {
    const payload = buildCreateAgentPayload(
      {
        organizationId: 'organization-id',
        name: 'agent-name',
      },
      'model-id',
    );

    expect(JSON.parse(JSON.stringify(payload))).toMatchObject({
      availability: AgentAvailability.INTERNAL,
    });
  });

  test('CreateAgent payload keeps caller availability override', () => {
    const payload = buildCreateAgentPayload(
      {
        organizationId: 'organization-id',
        name: 'agent-name',
        availability: AgentAvailability.PRIVATE,
      },
      'model-id',
    );

    expect(JSON.parse(JSON.stringify(payload))).toMatchObject({
      availability: AgentAvailability.PRIVATE,
    });
  });

  test('CreateAgent ConnectRPC JSON uses protobuf enum name', () => {
    const payload = buildCreateAgentRequestJson(
      {
        organizationId: 'organization-id',
        name: 'agent-name',
      },
      'model-id',
    );

    expect(JSON.parse(JSON.stringify(payload))).toMatchObject({
      availability: 'AGENT_AVAILABILITY_INTERNAL',
    });
  });

  test('CreateAgent ConnectRPC proto bytes include availability value', () => {
    const payload = buildCreateAgentRequestBytes(
      {
        organizationId: 'organization-id',
        name: 'agent-name',
      },
      'model-id',
    );

    expect(Buffer.from(payload).toString('hex')).toBe(
      '0a0a6167656e742d6e616d651209617373697374616e741a086d6f64656c2d6964420f6f7267616e697a6174696f6e2d69646801',
    );
  });
});
