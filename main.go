package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	openai "github.com/sashabaranov/go-openai"
)

// Variables used for command line parameters
var (
	TOKEN      string
	OPENAI_KEYS []string
	OPENAI_CLIENTS []*openai.Client
	PROMPT string
	ROUND_ROBIN_INDEX int
)

func init() {
	TOKEN = os.Getenv("TOKEN")
	OPENAI_KEYS = strings.Split(os.Getenv("OPENAI_KEY"), "|")
	PROMPT = os.Getenv("PROMPT")

	if TOKEN == "" {
		fmt.Println("Missing TOKEN env")
		os.Exit(1)
	}
	if OPENAI_KEYS == nil {
		fmt.Println("Missing OPENAI_KEYS env")
		os.Exit(1)
	}
	if PROMPT == "" {
		fmt.Println("Missing PROMPT env")
		os.Exit(1)
	}
	// fmt.Println(PROMPT)
}

func main() {
	openaiClient, err := setupOpenAI()
	if err != nil {
		fmt.Println("error creating OpenAI client,", err)
		return
	}
	OPENAI_CLIENTS = openaiClient

	discordSession, err := setupDiscordBot()
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Open a websocket connection to Discord and begin listening.
	err = discordSession.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	defer discordSession.Close()


	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

}

func setupDiscordBot() (s *discordgo.Session, err error) {
	fmt.Println("Setting up Discord bot...")
	s, err = discordgo.New("Bot " + TOKEN)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	s.AddHandler(responseHandler)
	// indent only receives message events.
	s.Identify.Intents = discordgo.IntentsGuildMessages
	fmt.Println("Done")
	return
}

func setupOpenAI() (c []*openai.Client, err error) {
	fmt.Println("Setting up OpenAI client...")
	for _, key := range OPENAI_KEYS {
		client := openai.NewClient(key)
		c = append(c, client)
	}
	// c = openai.NewClient(OPENAI_KEY)
	fmt.Println("Done")
	return
}

func responseHandler(sess *discordgo.Session, msg *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if msg.Author.ID == sess.State.User.ID {
		return
	}

	// Ignore all images message or icon or sticker
	if len(msg.Content) < 6 {
		return
	}

	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		// MaxTokens: 20,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: PROMPT,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: msg.Content,
			},
		},
		Stream: false,
	}

	roundRobin()
	client := OPENAI_CLIENTS[ROUND_ROBIN_INDEX]
	resp, err := client.CreateChatCompletion(
		context.Background(),
		req,
	)
	if err != nil {
		fmt.Printf("Completion error: %v\n", err)
		return
	}

	sess.ChannelMessageSendReply(msg.ChannelID, resp.Choices[0].Message.Content, msg.Reference())

}

func roundRobin() {
	ROUND_ROBIN_INDEX = (ROUND_ROBIN_INDEX + 1) % len(OPENAI_CLIENTS)
}
