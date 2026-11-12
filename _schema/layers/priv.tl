// Pulled from DrKLO 12.9.1 by cmd/layandroid.
// Client-only definitions missing from official layers.

---functions---

contacts.importCard#4fe196fe export_card:Vector<int> = User;

folders.deleteFolder#1c295881 folder_id:int = Updates;

messages.forwardMessage#33963bf9 peer:InputPeer id:int random_id:long = Updates;

messages.getAllChats#875f74be except_ids:Vector<long> = messages.Chats;

messages.getStatsURL#812c2ae6 flags:# dark:flags.0?true peer:InputPeer params:string = StatsURL;

messages.getWebViewResult#22b6c214 peer:InputPeer bot:InputUser query_id:long = messages.WebViewResult;

messages.reportReaction#61422a48 peer:InputPeer id:int user_id:InputUser = Bool;

messages.sendWebViewResultMessage#ddcf50eb query_id:long = WebViewMessageSent;

messages.setWebViewResult#e41cd11d query_id:long = Bool;

payments.requestRecurringPayment#146e958d user_id:InputUser recurring_init_charge:string invoice_media:InputMedia = Updates;

---types---

messages.webViewResult#aadf159b result:BotInlineResult users:Vector<User> = messages.WebViewResult;

