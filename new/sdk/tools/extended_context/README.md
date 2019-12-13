## Intro

ExtendedContext allows grafting two contexts together:
* One for cancelation/timeout/deadline
* One for Values

## Uses

When retrying a GRPC call, the original timers may be either expired or too short, but the Values
contained in the original context may still be relavent. 
