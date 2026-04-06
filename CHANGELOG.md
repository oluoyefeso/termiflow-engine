# Changelog

All notable changes to termiflow-engine will be documented in this file.

## [0.2.0] - 2026-04-06

### Added
- `taste` package for LLM-native personalization. Feed behavioral signals (reads, likes, skips), build taste profiles via exponential moving average, and wrap any LLMProvider for personalized scoring.
- `taste.UpdateProfile` builds user taste profiles from behavioral signals with cold-start alpha boost for faster early adaptation.
- `taste.PersonalizeScorer` wraps any LLMProvider to inject taste context into scoring prompts. Drop-in replacement for engine.NewCurator.
- `taste.Explain` generates natural-language explanations for why content scored the way it did.
- `taste.ProfileStore` interface with `FileProfileStore` (JSON files) and `InMemoryProfileStore` implementations.
- Signal kind constants: `SignalRead`, `SignalLike`, `SignalDislike`, `SignalSkip`, `SignalDwell`.
- Defensive value clamping and NaN/Inf protection in signal processing.
- Deep-copy of taste profiles in PersonalizeScorer to prevent data races.
- UserID validation to prevent path traversal in FileProfileStore.
- `examples/personalized-curation` demonstrating the full signal-to-profile-to-scoring pipeline.
- 46 tests covering types, stores, personalization, and explanation with edge cases.

## [0.1.0] - 2026-03-26

### Added
- Initial release: search, score, summarize, tag, curate pipeline.
- `LLMProvider` and `SearchProvider` interfaces.
- `StreamProvider` optional interface for streaming.
- `Curator` with configurable concurrency, thresholds, and scoring weights.
- `mock` package with deterministic test providers.
- 3 runnable examples (basic, custom-scoring, ask).
