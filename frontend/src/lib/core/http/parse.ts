// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

/** A response body did not match the expected shape at the fetch boundary. */
export class ApiParseError extends Error {
  readonly issues: readonly v.GenericIssue[];

  constructor(context: string, issues: [v.GenericIssue, ...v.GenericIssue[]]) {
    super(`Invalid ${context}: ${v.summarize(issues)}`);
    this.name = "ApiParseError";
    this.issues = issues;
  }
}

/** Validate an already-decoded payload against a schema. */
export function parsePayload<TSchema extends v.GenericSchema>(
  schema: TSchema,
  data: unknown,
  context = "server response",
): v.InferOutput<TSchema> {
  const result = v.safeParse(schema, data);
  if (!result.success) {
    throw new ApiParseError(context, result.issues);
  }
  return result.output;
}

/** Decode and validate the JSON body of a fetch response. */
export async function parseJson<TSchema extends v.GenericSchema>(
  schema: TSchema,
  response: Response,
  context = "server response",
): Promise<v.InferOutput<TSchema>> {
  return parsePayload(schema, await response.json(), context);
}
