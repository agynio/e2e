import { test, expect } from '@playwright/test';
import { buildCreateAgentPayload } from './console-api';

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
      availability: 'AGENT_AVAILABILITY_INTERNAL',
    });
  });

  test('CreateAgent payload keeps caller availability override', () => {
    const payload = buildCreateAgentPayload(
      {
        organizationId: 'organization-id',
        name: 'agent-name',
        availability: 'AGENT_AVAILABILITY_PRIVATE',
      },
      'model-id',
    );

    expect(JSON.parse(JSON.stringify(payload))).toMatchObject({
      availability: 'AGENT_AVAILABILITY_PRIVATE',
    });
  });
});
