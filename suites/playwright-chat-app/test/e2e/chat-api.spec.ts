import { test, expect } from '@playwright/test';
import { AgentAvailability } from '../../src/gen/agynio/api/agents/v1/agents_pb';
import { buildCreateAgentPayload } from './chat-api';

test.describe('chat api helpers', () => {
  test('CreateAgent payload omits default internal availability', () => {
    const payload = buildCreateAgentPayload({
      organizationId: 'organization-id',
      name: 'agent-name',
      role: 'assistant',
      model: 'model-id',
      description: 'description',
      configuration: '{}',
      image: 'alpine:3.21',
      initImage: 'ghcr.io/agynio/agent-init-codex:latest',
    });

    expect(JSON.parse(JSON.stringify(payload))).toEqual({
      organizationId: 'organization-id',
      name: 'agent-name',
      role: 'assistant',
      model: 'model-id',
      description: 'description',
      configuration: '{}',
      image: 'alpine:3.21',
      initImage: 'ghcr.io/agynio/agent-init-codex:latest',
    });
  });

  test('CreateAgent payload serializes private availability as enum number', () => {
    const payload = buildCreateAgentPayload({
      organizationId: 'organization-id',
      name: 'agent-name',
      role: 'assistant',
      model: 'model-id',
      description: 'description',
      configuration: '{}',
      image: 'alpine:3.21',
      initImage: 'ghcr.io/agynio/agent-init-codex:latest',
      availability: AgentAvailability.PRIVATE,
    });

    expect(JSON.parse(JSON.stringify(payload))).toMatchObject({
      availability: AgentAvailability.PRIVATE,
    });
  });
});
