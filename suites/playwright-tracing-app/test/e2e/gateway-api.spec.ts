import { test, expect } from '@playwright/test';
import { fromBinary } from '@bufbuild/protobuf';
import {
  AgentAvailability,
  AgentFinalMessage,
  CreateAgentRequestSchema,
} from '../../src/gen/agynio/api/agents/v1/agents_pb';
import { buildCreateAgentPayload, buildCreateAgentRequestBytes, buildCreateAgentRequestJson } from './gateway-api';

const createAgentOptions = {
  organizationId: 'organization-id',
  name: 'agent-name',
  role: 'assistant',
  model: 'model-id',
  description: 'description',
  configuration: '{}',
  image: 'alpine:3.21',
  environmentId: '00000000-0000-0000-0000-000000000001',
};

test.describe('tracing gateway api helpers', () => {
  test('CreateAgent payload sets internal availability enum', () => {
    const payload = buildCreateAgentPayload(createAgentOptions);

    expect(JSON.parse(JSON.stringify(payload))).toEqual({
      ...createAgentOptions,
      availability: AgentAvailability.INTERNAL,
      finalMessage: AgentFinalMessage.DEFAULT_THREAD,
    });
  });

  test('CreateAgent ConnectRPC JSON uses protobuf enum name', () => {
    const payload = buildCreateAgentRequestJson(createAgentOptions);

    expect(JSON.parse(JSON.stringify(payload))).toEqual({
      ...createAgentOptions,
      availability: 'AGENT_AVAILABILITY_INTERNAL',
      finalMessage: 'AGENT_FINAL_MESSAGE_DEFAULT_THREAD',
    });
  });

  // Decoded rather than compared to a hex literal. The literal encoded every
  // field of the request, so any change to the fixture had to be hand-computed
  // to keep a test that is only about how the enum is carried.
  test('CreateAgent ConnectRPC proto bytes include availability value', () => {
    const payload = buildCreateAgentRequestBytes(createAgentOptions);
    const request = fromBinary(CreateAgentRequestSchema, payload);

    expect(request.availability).toBe(AgentAvailability.INTERNAL);
  });
});
