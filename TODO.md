* ~~Load big dump in chunks~~
* ~~Support different languages (ru and en for a start)~~
* ~~Send pictures as a response~~
* ~~Handle all possible variants of queries (see slack/discord bots on Scryfall)~~
* ~~Handle multiple requests in same message~~
* Handle partial names
* EDHREC daily commander
* New spoilers
* Statistics for requests (who, what, when, etc.)
* Statistics for raw card pics added by users to channel
* Matchup stats
* piccache thread-safe


Bugs:
* Brazen borrower could not be found - name for double card seems to be smth like "Brazen Borrower // Petty Theft"
* when no card on mtgsale, price is 32.xxx
* fields.msg="[[Erebos, Bleak-Hearted]]" Could not sent reply ) ParseMode:MarkdownV2} due to error: Bad Request: can't parse entities: Character '-' is reserved and must be escaped with the preceding '\'
