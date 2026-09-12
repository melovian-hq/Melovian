---
name: no-ai-slop
description: "Rules and worked examples for writing prose that does not read like AI-generated slop. Consult before writing or editing any prose."
---

# No AI Slop

The prose rule list for this repo lives in `.agents/rules/prose.md`. This skill turns the rules that have worked examples into actionable guidance: each shows a WRONG version (the slop) and a RIGHT version (the fix). The pattern behind every fix is the same: replace the vague claim with a specific, checkable fact.

## Rule 1: No emdashes

The character is banned. Use a semicolon, a period, a comma, or restructure.

- WRONG: "The policy -- which affected millions -- was later reversed."
- RIGHT: "The policy affected millions of devices. The company reversed it in December 2017."

## Rule 4: No intensifiers

"Significantly", "dramatically", "extremely" and their kin are placeholders for evidence. Replace the word with the number it was standing in for.

- WRONG: "The pricing was significantly higher than the cost of the part."
- RIGHT: "They charged $1,200 for a repair that needed a $5 chip."

## Rule 5: No hollow statements

A sentence that asserts importance without a detail says nothing. End every claim on a concrete fact.

- WRONG: "This practice has had a significant impact on people."
- RIGHT: "The company replaced 11 million batteries in 2018, against the 1 to 2 million it had expected."

## Rule 7: No structural slop (repetitive layouts)

Three sections built from the same template read as machine output, even when each fact is true. Vary paragraph count, sentence rhythm, and how each section opens.

- WRONG (three sections, identical shape):
  ```
  In [year], [party] did [thing]. This affected [number] people. [Party] responded by [action].
  In [year], [party] did [thing]. This affected [number] people. [Party] responded by [action].
  In [year], [party] did [thing]. This affected [number] people. [Party] responded by [action].
  ```
- RIGHT (vary the shape):
  ```
  Section one: a detailed narrative with timeline and context across two paragraphs.
  Section two: a two-sentence summary, because the event is thinly documented.
  Section three: opens with the party's stated justification, then the contradicting evidence.
  ```

## Rule 11: No filler phrases

"In today's world", "It's important to note", "When it comes to" add length, not meaning. Open on the fact.

- WRONG: "In today's world, planned obsolescence affects many devices."
- RIGHT: "Apple, Samsung, and Google have each faced lawsuits alleging planned obsolescence."

## Rule 13: Write like a researcher, not a copywriter

If a sentence could sit on any advocacy or marketing site without changing a word, it is generic. Anchor it to something checkable.

- WRONG: "People deserve the right to repair their own devices."
- RIGHT: "The FTC voted 5-0 in July 2021 to step up enforcement against illegal repair restrictions."

## Rule 15: No weasel words

"May potentially", "can help to", "might be able to" hedge a claim into meaninglessness. Either the thing happens or it does not. Say which.

- WRONG: "Serialization may potentially prevent independent repair."
- RIGHT: "Replacing an iPhone 15 camera module without the manufacturer's calibration software disables optical image stabilization."

## Rule 16: No dramatic headings

A heading names what the section holds. It does not tease, dramatize, or abstract.

- WRONG: "The Hidden Cost of Planned Obsolescence"
- RIGHT: "Economic impact of shortened product lifespans"

## Rule 19: No fabricated attributions

Never put a position in a named person's mouth from inference. State only what they actually did or said, with the real source.

- WRONG: "Senator Smith has argued that the right to repair is essential."
- RIGHT: "Senator Smith co-sponsored the Fair Repair Act in January 2024."

## Root-cause differentiation

When you contrast two things, name the concrete difference that separates them. Do not assert that one is exempt, newer, better, or unaffected without saying what specifically makes it so.

- WRONG: "2020+ Leaf models are unaffected and use the MyNISSAN app instead."
- RIGHT: "2020+ Leaf models shipped with 4G/LTE telematics units connected to a newer cloud platform, replacing the 2G/3G units in earlier models. Those vehicles use the MyNISSAN app, which talks to a different backend."

Whenever you say A differs from B, name the part, the version, the date, the mechanism, or the supply-chain change that makes the difference real. If you do not have that detail, do not imply the difference exists.

## Newer patterns (2025-2026 model output)

The classic tells (delve, em dashes, "in today's world") are filtered out of newer models. Current output fails in different places. Each of these has a fuller list in `references/ai-writing-detection.md`.

### The -ing tail

AI attaches a present-participle clause to fake analysis: ", highlighting the importance of...", ", contributing to the ecosystem...", ", ensuring a seamless experience...". The clause adds an assertion, not a fact.

- WRONG: "The station added eight platforms, contributing to the socio-economic development of the region."
- RIGHT: "The station added eight platforms in 2019. Freight volume rose 40% by 2023."

Fix: delete the tail or replace it with the fact it was gesturing at.

### Copula avoidance

AI avoids "is/are/has" and substitutes elevated verbs: serves as, stands as, functions as, represents, boasts, features, offers, maintains, "holds the distinction of being". Write is/has/does.

- WRONG: "The player boasts a robust queue and serves as the primary playback interface."
- RIGHT: "The player is the primary playback interface. Its queue persists across restarts."

### Vague attribution

Opinions get attributed to nameless authorities: "experts say", "critics argue", "observers note", "industry reports", "studies show". If you cannot name who holds the opinion, the opinion does not go in.

- WRONG: "Experts believe the change improves playback reliability."
- RIGHT: "The 14 failed-gapless reports in issue 2341 stopped after the patch."

### Formulaic endings

AI ends articles with "Despite its success, X faces challenges..." or "the future looks promising". End on the last concrete fact instead.

- WRONG: "Despite these challenges, the project continues to evolve and remains well positioned for the future."
- RIGHT: "Two issues remain open: gapless playback on Ogg Vorbis and cover art over 2 MB."

### Chatbot residue

Anything addressed to the user as a chat partner does not belong in docs, comments, commit messages, or UI strings: "Great question!", "Hope this helps!", "Let me know if...", "Here's the thing:", "Picture this:", "But here's the kicker:", "Let that sink in.", "Read that again.", "Sound familiar?", "Want to know the best part?". Also banned: knowledge-cutoff narration ("as of my last update", "based on available information", "while specific details are limited") and leftover placeholders ("[Your Name]", "2025-XX-XX", "Delete this section before submission").

- WRONG: "Here's the thing: the cache is not just fast, it's also memory-efficient. Let that sink in."
- RIGHT: "The cache holds 10,000 entries in under 40 MB."

### Rule of three

Adjective stacks and triple-phrase rhythms simulate completeness: "fast, reliable, and secure". Keep the one property that is real and measurable.

- WRONG: "The new engine is fast, reliable, and secure."
- RIGHT: "The new engine cut p99 seek latency from 40 ms to 12 ms in benchmarks."

### Marketing and dev-marketing register

Landing-page copy does not belong in prose: unlock, elevate, supercharge, game-changer, best-in-class, world-class, "the future of X", "designed to", blazing fast, buttery smooth, under the hood, battle-tested, production-ready, rock-solid, seamless integration, it just works, zero-config. Replace each with the metric, the mechanism, or the actual requirement.

- WRONG: "Melovian is a blazing-fast, rock-solid player that just works out of the box."
- RIGHT: "Melovian indexes 50k local tracks in about 20 seconds and plays while scanning continues."

### Formatting bleed

Chat formatting habits that mark generated output: bold-colon bullets ("**Label:** text") as the default list shape, boldface on every key term, emoji in headings, Title Case headings, "---" breaks between every section, "Key takeaways" blocks, and list preambles like "Here are the main features:". Match the file's existing conventions instead.

## Self-check before returning text

Run this pass on every piece of prose before you hand it back. The full banned lists are in `references/ai-writing-detection.md`; check against them directly.

1. Search for the emdash character, " -- ", and en-dash stand-ins. Remove every one (Rule 1).
2. Scan for banned verbs (delve, leverage, utilize, foster, bolster, underscore, unveil, streamline, showcase, garner, align with, ensuring) and replace with plain equivalents.
3. Scan for banned adjectives and intensifiers (robust, comprehensive, pivotal, seamless, significantly, extremely, truly, vibrant, meticulous, game-changing, best-in-class) and cut or replace.
4. Scan for banned transitions and openers (Furthermore, Moreover, That being said, Additionally, In today's world, It's worth noting that).
5. Check every number: is it real and attributable? If not, cut it (Rule 2).
6. Check every sentence ends on a concrete detail, not an assertion of importance (Rule 5).
7. Check headings: does each name the content rather than tease it (Rule 16)?
8. Check for repeated points and repeated section shapes (Rules 6, 7).
9. Count hedging markers per paragraph. More than three is a red flag.
10. Read it aloud. If a phrase would sound unnatural to a colleague, rewrite it.
11. Grep for -ing tails: ", highlighting", ", underscoring", ", reflecting", ", contributing", ", ensuring", ", fostering", ", shaping", ", positioning", ", cementing". Cut or replace with a fact.
12. Grep for copula avoidance: "serves as", "stands as", "functions as", "represents", "boasts", "features", "holds the distinction". Rewrite to is/has/does.
13. Grep for unnamed attribution: "experts", "critics", "observers", "sources", "studies show". Name them or cut.
14. Grep for chatbot residue: "Hope this helps", "let me know", "Great question", "Here's the thing", "Let that sink in", "as of my last", bracketed placeholders. Delete the sentence.
15. Check the ending: if it is a challenges/future-outlook formula or balanced-on-command optimism, end on a fact.
16. Normalize unicode: no non-breaking spaces, zero-width characters, or smart-quote drift from the file's convention.
