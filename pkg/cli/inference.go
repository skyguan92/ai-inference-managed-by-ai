package cli

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/spf13/cobra"
)

func NewInferenceCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inference",
		Short: "Inference commands",
		Long: `Run inference operations on AI models.

This includes chat completions, text embeddings, and other inference tasks.`,
	}

	cmd.AddCommand(NewInferenceChatCommand(root))
	cmd.AddCommand(NewInferenceEmbedCommand(root))

	return cmd
}

func NewInferenceChatCommand(root *RootCommand) *cobra.Command {
	var (
		model       string
		message     string
		temperature float64
		maxTokens   int
		stream      bool
	)

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Run chat completion",
		Long: `Run a chat completion request against a model.

Messages can be provided via the --message flag or piped via stdin.`,
		Example: `  # Simple chat
  aima inference chat --model llama3.2 --message "Hello, how are you?"

  # With temperature
  aima inference chat --model llama3.2 --message "Hello" --temperature 0.7

  # With max tokens
  aima inference chat --model llama3.2 --message "Hello" --max-tokens 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInferenceChat(cmd.Context(), root, model, message, temperature, maxTokens, stream)
		},
	}

	cmd.Flags().StringVarP(&model, "model", "m", "", "Model name (required)")
	cmd.Flags().StringVarP(&message, "message", "M", "", "Chat message")
	cmd.Flags().Float64Var(&temperature, "temperature", 0.7, "Temperature for sampling")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 0, "Maximum tokens to generate")
	cmd.Flags().BoolVar(&stream, "stream", false, "Stream response (not yet supported)")

	_ = cmd.MarkFlagRequired("model")

	return cmd
}

func runInferenceChat(ctx context.Context, root *RootCommand, model, message string, temperature float64, maxTokens int, stream bool) error {
	if model == "" {
		return fmt.Errorf("model is required")
	}

	if message == "" {
		return fmt.Errorf("message is required")
	}

	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": message},
		},
		"temperature": temperature,
	}

	if maxTokens > 0 {
		input["max_tokens"] = maxTokens
	}

	if stream {
		return fmt.Errorf("streaming is not yet supported")
	}

	req := &gateway.Request{
		Type:  gateway.TypeCommand,
		Unit:  "inference.chat",
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		errMsg := fmt.Sprintf("%s: %s", resp.Error.Code, resp.Error.Message)
		if resp.Error.Details != nil {
			errMsg = fmt.Sprintf("%s\ndetails: %v", errMsg, resp.Error.Details)
		}
		PrintError(fmt.Errorf("%s", errMsg), opts)
		return fmt.Errorf("chat completion failed: %v", resp.Error.Details)
	}

	return PrintOutput(resp.Data, opts)
}

func NewInferenceEmbedCommand(root *RootCommand) *cobra.Command {
	var (
		model string
		input string
	)

	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Generate text embeddings",
		Long:  `Generate embeddings for text using an embedding model.`,
		Example: `  # Generate embedding
  aima inference embed --model nomic-embed-text --input "Hello, world!"

  # With JSON output
  aima inference embed --model nomic-embed-text --input "Hello" --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInferenceEmbed(cmd.Context(), root, model, input)
		},
	}

	cmd.Flags().StringVarP(&model, "model", "m", "", "Embedding model name (required)")
	cmd.Flags().StringVarP(&input, "input", "i", "", "Text to embed (required)")

	_ = cmd.MarkFlagRequired("model")
	_ = cmd.MarkFlagRequired("input")

	return cmd
}

func runInferenceEmbed(ctx context.Context, root *RootCommand, model, inputText string) error {
	if model == "" {
		return fmt.Errorf("model is required")
	}

	if inputText == "" {
		return fmt.Errorf("input is required")
	}

	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "inference.embed",
		Input: map[string]any{
			"model": model,
			"input": inputText,
		},
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("embedding generation failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}
