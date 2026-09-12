// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import axe from "axe-core";
import { expect } from "vitest";

// Rules that cannot run faithfully in jsdom. color-contrast needs real
// computed paint colors and region misreports landmarks in synthetic
// fragments that are not a full page.
const DISABLED_RULES: Record<string, { enabled: boolean }> = {
  "color-contrast": { enabled: false },
  region: { enabled: false },
};

export function formatA11yViolations(violations: axe.Result[]): string {
  return violations
    .map((violation) => {
      const nodes = violation.nodes
        .map(
          (node) =>
            `    ${node.target.join(" ")}\n      ${node.failureSummary ?? ""}`,
        )
        .join("\n");
      return `  [${violation.impact ?? "unknown"}] ${violation.id}: ${violation.help}\n${violation.helpUrl}\n${nodes}`;
    })
    .join("\n\n");
}

export async function runA11yScan(
  element: axe.ElementContext = document.body,
): Promise<axe.AxeResults> {
  return axe.run(element, { rules: DISABLED_RULES });
}

export async function expectNoA11yViolations(
  element: axe.ElementContext = document.body,
): Promise<void> {
  const results = await runA11yScan(element);
  if (results.violations.length > 0) {
    console.error(formatA11yViolations(results.violations));
  }
  expect(results.violations.map((violation) => violation.id)).toEqual([]);
}

// Bits UI portals mount under document.body and can outlive unmount while
// close animations settle. Sweep leftover nodes between tests so axe does
// not scan stale markup.
export function removeLeftoverPortalNodes(): void {
  for (const child of Array.from(document.body.children)) {
    child.remove();
  }
}
