package engine

// Default prompt templates used by the engine.
// Override the scoring prompt via WithScoringPrompt().

const (
	// DefaultScoringPrompt evaluates content relevance to a topic.
	// Placeholder: %s = topic, %s = title, %s = snippet.
	DefaultScoringPrompt = `You are evaluating if a piece of content is relevant to a user's topic subscription.

Topic: %s
Content Title: %s
Content Snippet: %s

Rate the relevance from 0.0 to 1.0 where:
- 0.0-0.3: Not relevant
- 0.4-0.6: Somewhat relevant
- 0.7-0.9: Highly relevant
- 1.0: Perfectly relevant

Respond with only a number between 0.0 and 1.0.`

	// DefaultSummaryPrompt generates a concise summary for a topic subscriber.
	// Placeholder: %s = topic, %s = title, %s = content.
	DefaultSummaryPrompt = `Summarize the following article in 2-3 sentences for a developer interested in "%s".
Focus on the key technical insights and why it matters.

Title: %s
Content: %s

Summary:`

	// DefaultTagPrompt extracts technical tags from content.
	// Placeholder: %s = title, %s = content.
	DefaultTagPrompt = `Extract 2-4 relevant technical tags from this content. Return only lowercase tags separated by commas.

Title: %s
Content: %s

Tags:`

	// DefaultAskSystemPrompt is the system message for question-answering.
	DefaultAskSystemPrompt = `You are a helpful assistant that provides accurate, well-researched answers. Use the provided sources to inform your response. Be concise but thorough.`
)
