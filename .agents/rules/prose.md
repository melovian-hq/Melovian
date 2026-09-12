# Prose rules

These rules apply to user-facing text, comments, docs, and commit messages.

## Banned characters and symbols

- No emdashes, en dash stand-ins, or " -- " substitutes. Use a period, comma, or parentheses.
- No emojis or emoji arrows.
- No backtick code in code comments. Use prose or code references in the surrounding text.
- Minimal use of semicolons, backticks, or other professional writing styles in prose.
- No non-breaking spaces, zero-width characters, or smart quotes that drift from the file's convention.

## Banned words and patterns

- No intensifiers: "significantly", "dramatically", "extremely", "robustly", "seamlessly". Replace with a number or concrete detail.
- No hollow statements that assert importance without a checkable fact. End every claim on a specific detail.
- No filler phrases: "In today's world", "It is important to note", "When it comes to".
- No weasel words: "may potentially", "can help to", "might be able to".
- No dramatic headings that tease. A heading names the content. Sentence case, not Title Case.
- No -ing participle tails that fake analysis: ", highlighting", ", underscoring", ", contributing to", ", ensuring", ", reflecting". Cut the tail or state the fact.
- No copula avoidance: "serves as", "stands as", "boasts", "features" where "is" or "has" is the truth.
- No unnamed attribution: "experts say", "critics argue", "studies show". Name them or cut the claim.
- No chatbot residue: "Here's the thing", "Let that sink in", "Hope this helps", "Great question", placeholders, "as of my last update".
- No rule-of-three adjective stacks: "fast, reliable, and secure". Keep the one measurable claim.
- No marketing register: "unlock", "supercharge", "game-changer", "best-in-class", "blazing fast", "it just works", "seamless integration". Name the metric or mechanism.
- No formulaic endings: "challenges and future prospects" sections, "despite these challenges", "the future looks bright". End on a fact.

## Writing style

- Write like a researcher, not a copywriter. Anchor every claim to a source, number, version, or date.
- No fabricated attributions. State only what a named person or organization actually did or said.
- Name the concrete difference when you contrast two things. Do not say one is "better" or "newer" without saying what makes it so.
- When in doubt, read `.agents/skills/no-ai-slop/SKILL.md`. The full banned-word lists live in `.agents/skills/no-ai-slop/references/ai-writing-detection.md`.
