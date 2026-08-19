import { test, expect } from '@playwright/test';
import { fromBinary } from '@bufbuild/protobuf';
import {
  AgentAvailability,
  AgentFinalMessage,
  CreateAgentRequestSchema,
} from '../../src/gen/agynio/api/agents/v1/agents_pb';
import {
  buildCreateAgentPayload,
  buildCreateAgentRequestBytes,
  buildCreateAgentRequestJson,
} from './chat-api';

// Named rather than derived: buildCreateAgentPayload invents a nickname when
// none is given, with a random suffix, so a payload built from options without
// one can never equal those options.
const createAgentOptions = {
  organizationId: 'organization-id',
  name: 'agent-name',
  nickname: 'agent-nickname',
  role: 'assistant',
  model: 'model-id',
  description: 'description',
  configuration: '{}',
  image: 'alpine:3.21',
  environmentId: '00000000-0000-0000-0000-000000000001',
};

test.describe('chat api helpers', () => {
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
