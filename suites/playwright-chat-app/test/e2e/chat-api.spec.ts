import { test, expect } from '@playwright/test';
import { AgentAvailability } from '../../src/gen/agynio/api/agents/v1/agents_pb';
import { buildCreateAgentPayload } from './chat-api';

test.describe('chat api helpers', () => {
  test('CreateAgent payload includes internal availability', () => {
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

    expect(payload.availability).toBe(AgentAvailability.INTERNAL);
    expect(JSON.parse(JSON.stringify(payload))).toMatchObject({
      availability: AgentAvailability.INTERNAL,
    });
  });
});
