import { test, expect } from '@playwright/test';
import { fromBinary } from '@bufbuild/protobuf';
import { AgentAvailability, CreateAgentRequestSchema } from '../../src/gen/agynio/api/agents/v1/agents_pb';
import {
  buildCreateAgentPayload,
  buildCreateAgentRequestBytes,
  buildCreateAgentRequestJson,
} from './chat-api';

const createAgentOptions = {
  organizationId: 'organization-id',
  name: 'agent-name',
  role: 'assistant',
  model: 'model-id',
  description: 'description',
  configuration: '{}',
  image: 'alpine:3.21',
  initImage: 'ghcr.io/agynio/agent-init-codex:0.13.29',
};

test.describe('chat api helpers', () => {
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
    const request = fromBinary(CreateAgentRequestSchema, payload);

    expect(request.availability).toBe(AgentAvailability.INTERNAL);
  });

  test('CreateAgent payload serializes private availability enum', () => {
    const payload = buildCreateAgentPayload({
      ...createAgentOptions,
      availability: AgentAvailability.PRIVATE,
    });

    expect(JSON.parse(JSON.stringify(payload))).toMatchObject({
      availability: AgentAvailability.PRIVATE,
    });
  });

  test('CreateAgent ConnectRPC JSON uses private protobuf enum name', () => {
    const payload = buildCreateAgentRequestJson({
      ...createAgentOptions,
      availability: AgentAvailability.PRIVATE,
    });

    expect(JSON.parse(JSON.stringify(payload))).toMatchObject({
      availability: 'AGENT_AVAILABILITY_PRIVATE',
    });
  });

  test('CreateAgent rejects unsupported availability enum', () => {
    expect(() =>
      buildCreateAgentPayload({
        ...createAgentOptions,
        availability: AgentAvailability.UNSPECIFIED,
      }),
    ).toThrow('Unsupported agent availability: 0');
  });
});
