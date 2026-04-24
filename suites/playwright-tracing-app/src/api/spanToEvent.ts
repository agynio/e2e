import type { AnyValue, KeyValue } from '@/gen/opentelemetry/proto/common/v1/common_pb';
import type { ResourceSpans, Span } from '@/gen/opentelemetry/proto/trace/v1/trace_pb';
import { Status_StatusCode } from '@/gen/opentelemetry/proto/trace/v1/trace_pb';
import type { RunEventStatus, RunEventType, RunTimelineEvent } from '@/api/types/agents';

type FlattenedSpan = { span: Span; resourceAttrs: KeyValue[] };

export const SPAN_NAME_TO_EVENT_TYPE: Record<string, RunEventType> = {
  'invocation.message': 'invocation_message',
  injection: 'injection',
  'llm.call': 'llm_call',
  'tool.execution': 'tool_execution',
  summarization: 'summarization',
};

export function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

export function hexToBytes(hex: string): Uint8Array {
  const trimmed = hex.trim();
  const normalized = trimmed.startsWith('0x') ? trimmed.slice(2) : trimmed;
  if (normalized.length % 2 !== 0) {
    throw new Error(`Invalid hex string length: ${hex}`);
  }
  const bytes = new Uint8Array(normalized.length / 2);
  for (let i = 0; i < bytes.length; i += 1) {
    const byte = Number.parseInt(normalized.slice(i * 2, i * 2 + 2), 16);
    if (Number.isNaN(byte)) {
      throw new Error(`Invalid hex string: ${hex}`);
    }
    bytes[i] = byte;
  }
  return bytes;
}

function findAttr(attrs: KeyValue[], key: string): AnyValue | undefined {
  return attrs.find((attr) => attr.key === key)?.value;
}

export function getStringAttr(attrs: KeyValue[], key: string): string | null {
  const value = findAttr(attrs, key);
  if (!value || value.value.case !== 'stringValue') return null;
  return value.value.value;
}

export function getIntAttr(attrs: KeyValue[], key: string): number | null {
  const value = findAttr(attrs, key);
  if (!value || value.value.case !== 'intValue') return null;
  return Number(value.value.value);
}

export function getDoubleAttr(attrs: KeyValue[], key: string): number | null {
  const value = findAttr(attrs, key);
  if (!value || value.value.case !== 'doubleValue') return null;
  return value.value.value;
}

export function getJsonAttr(attrs: KeyValue[], key: string): unknown {
  const raw = getStringAttr(attrs, key);
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch (error) {
    console.warn(`Failed to parse JSON attribute: ${key}`, { error, raw });
    return null;
  }
}

export function deriveEventStatus(span: Span): RunEventStatus {
  if (span.status?.code === Status_StatusCode.ERROR) return 'error';
  if (span.endTimeUnixNano === 0n) return 'running';
  return 'success';
}

export function nanosToIso(nanos: bigint): string {
  if (nanos === 0n) return '';
  return new Date(Number(nanos / 1_000_000n)).toISOString();
}

export function nanosToDurationMs(start: bigint, end: bigint): number | null {
  if (end === 0n) return null;
  return Number((end - start) / 1_000_000n);
}

function mapSpanNameToEventType(name: string): RunEventType {
  const type = SPAN_NAME_TO_EVENT_TYPE[name];
  if (!type) {
    throw new Error(`Unhandled span name: ${name}`);
  }
  return type;
}

type ToolCallPayload = {
  call_id: string;
  name: string;
  arguments?: unknown;
};

function isToolCallPayload(value: unknown): value is ToolCallPayload {
  if (!value || typeof value !== 'object') return false;
  const record = value as Record<string, unknown>;
  return typeof record.call_id === 'string' && typeof record.name === 'string';
}

function parseToolCalls(raw: string | null): Array<{ callId: string; name: string; arguments: unknown }> {
  if (!raw) return [];
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`Invalid tool calls JSON: ${message}`, { cause: error });
  }
  if (!Array.isArray(parsed)) {
    throw new Error('Invalid tool calls payload: expected an array');
  }
  return parsed.map((toolCall, index) => {
    if (!isToolCallPayload(toolCall)) {
      throw new Error(`Invalid tool call payload at index ${index}`);
    }
    return {
      callId: toolCall.call_id,
      name: toolCall.name,
      arguments: toolCall.arguments ?? null,
    };
  });
}

function extractLlmCall(span: Span) {
  const attrs = span.attributes;
  const usage = {
    inputTokens: getIntAttr(attrs, 'gen_ai.usage.input_tokens'),
    cachedInputTokens: getIntAttr(attrs, 'gen_ai.usage.cache_read.input_tokens'),
    outputTokens: getIntAttr(attrs, 'gen_ai.usage.output_tokens'),
    reasoningTokens: getIntAttr(attrs, 'agyn.usage.reasoning_tokens'),
    totalTokens: null as number | null,
  };
  if (usage.inputTokens != null && usage.outputTokens != null) {
    usage.totalTokens = usage.inputTokens + usage.outputTokens;
  }

  return {
    provider: getStringAttr(attrs, 'gen_ai.system'),
    model: getStringAttr(attrs, 'gen_ai.request.model'),
    temperature: getDoubleAttr(attrs, 'gen_ai.request.temperature'),
    topP: getDoubleAttr(attrs, 'gen_ai.request.top_p'),
    stopReason: getStringAttr(attrs, 'gen_ai.response.finish_reason'),
    inputContextItems: [],
    contextDeltaStatus: 'unknown' as const,
    responseText: getStringAttr(attrs, 'agyn.llm.response_text'),
    rawResponse: getJsonAttr(attrs, 'agyn.llm.raw_response'),
    toolCalls: parseToolCalls(getStringAttr(attrs, 'agyn.llm.tool_calls')),
    usage,
  };
}

function extractToolExecution(span: Span) {
  const attrs = span.attributes;
  const status = deriveEventStatus(span);
  return {
    toolName: getStringAttr(attrs, 'agyn.tool.name') ?? '',
    toolCallId: getStringAttr(attrs, 'agyn.tool.call_id'),
    execStatus: (status === 'error' ? 'error' : 'success') as 'success' | 'error',
    input: getJsonAttr(attrs, 'agyn.tool.input'),
    output: getJsonAttr(attrs, 'agyn.tool.output'),
    errorMessage: span.status?.message || getStringAttr(attrs, 'error.type') || null,
    raw: null,
  };
}

function extractSummarization(span: Span) {
  const attrs = span.attributes;
  return {
    summaryText: getStringAttr(attrs, 'agyn.summarization.text') ?? '',
    newContextCount: getIntAttr(attrs, 'agyn.summarization.new_context_count') ?? 0,
    oldContextTokens: getIntAttr(attrs, 'agyn.summarization.old_context_tokens'),
    raw: null,
  };
}

function extractMessage(span: Span) {
  const attrs = span.attributes;
  return {
    messageId: bytesToHex(span.spanId),
    role: getStringAttr(attrs, 'agyn.message.role') ?? 'user',
    kind: getStringAttr(attrs, 'agyn.message.kind'),
    text: getStringAttr(attrs, 'agyn.message.text'),
    source: null,
    createdAt: nanosToIso(span.startTimeUnixNano),
  };
}

export function spanToEvent(span: Span, resourceAttrs?: KeyValue[]): RunTimelineEvent {
  const attrs = span.attributes;
  const spanName = span.name;
  const status = deriveEventStatus(span);
  const spanIdHex = bytesToHex(span.spanId);
  const traceIdHex = bytesToHex(span.traceId);

  const type = mapSpanNameToEventType(spanName);

  const base: RunTimelineEvent = {
    id: spanIdHex,
    runId: traceIdHex,
    threadId: getStringAttr(resourceAttrs ?? [], 'agyn.thread.id') ?? '',
    type,
    status,
    ts: nanosToIso(span.startTimeUnixNano),
    startedAt: nanosToIso(span.startTimeUnixNano),
    endedAt: span.endTimeUnixNano === 0n ? null : nanosToIso(span.endTimeUnixNano),
    durationMs: nanosToDurationMs(span.startTimeUnixNano, span.endTimeUnixNano),
    nodeId: null,
    sourceKind: 'tracing',
    sourceSpanId: spanIdHex,
    metadata: null,
    errorCode: null,
    errorMessage: status === 'error'
      ? (span.status?.message || getStringAttr(attrs, 'error.type')) ?? null
      : null,
    attachments: [],
  };

  switch (type) {
    case 'llm_call':
      base.llmCall = extractLlmCall(span);
      break;
    case 'tool_execution':
      base.toolExecution = extractToolExecution(span);
      break;
    case 'summarization':
      base.summarization = extractSummarization(span);
      break;
    case 'invocation_message':
      base.message = extractMessage(span);
      break;
  }

  return base;
}

export function flattenResourceSpans(resourceSpans: ResourceSpans[]): FlattenedSpan[] {
  const flattened: FlattenedSpan[] = [];
  for (const resourceSpan of resourceSpans) {
    const resourceAttrs = resourceSpan.resource?.attributes ?? [];
    for (const scopeSpan of resourceSpan.scopeSpans) {
      for (const span of scopeSpan.spans) {
        flattened.push({ span, resourceAttrs });
      }
    }
  }
  return flattened;
}
