// Pulled from DrKLO 12.9.1 by cmd/layandroid.
// Client-only definitions missing from official layers.

---functions---

account.createTheme#8432c21f flags:# slug:string title:string document:flags.2?InputDocument settings:flags.3?InputThemeSettings = Theme;

account.getTheme#8d9d742b format:string theme:InputTheme document_id:long = Theme;

account.installTheme#7ae43737 flags:# dark:flags.0?true format:flags.1?string theme:flags.1?InputTheme = Bool;

account.registerDevice#637ea878 token_type:int token:string = Bool;

account.updateTheme#5cb367d5 flags:# format:string theme:InputTheme slug:flags.0?string title:flags.1?string document:flags.2?InputDocument settings:flags.3?InputThemeSettings = Theme;

auth.signUp#80eee427 phone_number:string phone_code_hash:string first_name:string last_name:string = auth.Authorization;

bots.exportBotToken#63b089 bot_id:long revoke:Bool = bots.ExportedBotToken;

channels.editCreator#8f38cd1f channel:InputChannel user_id:InputUser password:InputCheckPasswordSRP = Updates;

channels.getMessages#93d7b347 channel:InputChannel id:Vector<int> = messages.Messages;

channels.inviteToChannel#199f3a6c channel:InputChannel users:Vector<InputUser> = messages.InvitedUsers;

channels.joinChannel#24b524c5 channel:InputChannel = messages.ChatInviteJoinResult;

contacts.importCard#4fe196fe export_card:Vector<int> = User;

contacts.search#11f812d8 q:string limit:int = contacts.Found;

folders.deleteFolder#1c295881 folder_id:int = Updates;

langpack.getLangPack#9ab5c58e lang_code:string = LangPackDifference;

messages.createChat#34a818 flags:# users:Vector<InputUser> title:string ttl_period:flags.0?int = messages.InvitedUsers;

messages.forwardMessage#33963bf9 peer:InputPeer id:int random_id:long = Updates;

messages.forwardMessages#41d41ade flags:# silent:flags.5?true background:flags.6?true with_my_score:flags.8?true drop_author:flags.11?true drop_media_captions:flags.12?true noforwards:flags.14?true from_peer:InputPeer id:Vector<int> random_id:Vector<long> to_peer:InputPeer top_msg_id:flags.9?int reply_to:flags.22?InputReplyTo schedule_date:flags.10?int schedule_repeat_period:flags.24?int send_as:flags.13?InputPeer quick_reply_shortcut:flags.17?InputQuickReplyShortcut video_timestamp:flags.20?int allow_paid_stars:flags.21?long suggested_post:flags.23?SuggestedPost = Updates;

messages.getAllChats#875f74be except_ids:Vector<long> = messages.Chats;

messages.getMessages#4222fa74 id:Vector<int> = messages.Messages;

messages.getStatsURL#812c2ae6 flags:# dark:flags.0?true peer:InputPeer params:string = StatsURL;

messages.getWebViewResult#22b6c214 peer:InputPeer bot:InputUser query_id:long = messages.WebViewResult;

messages.importChatInvite#6c50051c hash:string = messages.ChatInviteJoinResult;

messages.reportReaction#61422a48 peer:InputPeer id:int user_id:InputUser = Bool;

messages.sendWebViewResultMessage#ddcf50eb query_id:long = WebViewMessageSent;

messages.setWebViewResult#e41cd11d query_id:long = Bool;

messages.uploadMedia#519bc2b1 peer:InputPeer media:InputMedia = MessageMedia;

payments.requestRecurringPayment#146e958d user_id:InputUser recurring_init_charge:string invoice_media:InputMedia = Updates;

updates.getDifference#25939651 flags:# pts:int pts_total_limit:flags.0?int date:int qts:int = updates.Difference;

---types---

langpack.getLanguages#800fd57d = langpack.GetLanguages;

langpack.getStrings#2e1ee318 lang_code:string keys:Vector<string> = langpack.GetStrings;

messages.webViewResult#aadf159b result:BotInlineResult users:Vector<User> = messages.WebViewResult;

