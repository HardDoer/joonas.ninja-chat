package main

// EventTyping - An event which implies that the user is currently typing.
const EventTyping = "typing"

// EventMessage - An event which contains a chat message.
const EventMessage = "message"

// EventJoin - An event which is sent when the user joins the chat.
const EventJoin = "join"

// EventLogin - An event which is sent when user wishes to log in.
const EventLogin = "login"

// EventTokenRefresh - An event which is sent when the token is refreshed and a new token is sent back to the user.
const EventTokenRefresh = "tokenRefresh"

// EventNameChange - An event which contains information that the user changed their name and also the new name.
const EventNameChange = "nameChange"

// EventNotification - A general notification event. Server status etc.
const EventNotification = "notification"

// EventNotification - A general notification event. Server status etc.
const EventErrorNotification = "errorNotification"

// EventChatHistory - An event that contains the previous chathistory.
const EventChatHistory = "chatHistory"

// EventWho - An event for who
const EventWho = "whoCommand"

// EventWho - An event for help
const EventHelp = "helpCommand"

const EventChannelList = "channelList"

const PublicChannelName = "public"

// CommandWhereAmI - What channel are you on.
const CommandWhereAmI = "where"

// CommandNick - Set your nickname
const CommandNameChange = "nick"

// CommandWho - List users command for the chat.
const CommandWho = "who"

// CommandUser - Command for user related operations.
const CommandUser = "user"

// CommandChannel - Channel command that is used as a prefix for all the different channel operations like create and list.
const CommandChannel = "channel"

const CommandHelp = "help"

const SystemName = ""

const ErrorCodeCommandNotRecognized = "0"
