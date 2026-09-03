# 🎙️ TTS Models & Voices Catalog

This document contains the complete list of conversational AI models and the catalog of Text-to-Speech voices supported by the **WhatsApp AI Call Assistant Bot**.

---

## 🤖 1. Gemini AI Models (`GEMINI_MODEL`)

The Google Gemini conversational AI models used to generate the bot's phone-call replies:

| Model Name | Status | Latency | Recommended Use |
| :--- | :--- | :--- | :--- |
| **`gemini-3.1-flash-lite`** | ⭐ **Default** | **~0.3 seconds** | **Highly recommended!** Fastest, token-efficient, and free of tight daily quotas. |
| **`gemini-3.6-flash`** | Newest version | ~0.5 seconds | Latest generation model with strong reasoning ability. |
| **`gemini-3.5-flash`** | Stable | ~0.6 seconds | A stable alternative Flash model. |
| **`gemini-flash-latest`** | Fallback | ~0.7 seconds | Used automatically if the primary model fails. |

---

## 🔊 2. Google Gemini Native TTS (`TTS_ENGINE=geminitts`)

Speech synthesized natively by Google Gemini Audio. It produces voices with natural intonation, soft breathing, and a lively conversational delivery.

- **Requirement**: Uses your existing `GEMINI_API` key (100% free).
- **Automatic fallback**: If the official Google Gemini TTS API errors out or runs out of quota, the bot automatically switches to the **NexRay Gemini TTS API fallback** (`https://api.nexray.eu.cc/ai/gemini-tts?text=...`), and then to **Edge TTS**, so the phone conversation is never interrupted.

| Voice Name (`TTS_VOICE`) | Gender | Voice Character |
| :--- | :--- | :--- |
| **`Puck`** ⭐ *(Default)* | Male / Neutral | Warm, friendly, and conversational in tone. |
| **`Kore`** | Female | Soft, smooth, and calm. |
| **`Aoede`** | Female | Very expressive, emotional, and lively. |
| **`Charon`** | Male | Deep, authoritative, and calm. |
| **`Fenrir`** | Male | Energetic, enthusiastic, and assertive. |

---

## ⚡ 3. Microsoft Edge Neural TTS (`TTS_ENGINE=edgetts`)

*Microsoft Azure Neural Speech* synthesis. **100% FREE FOREVER** — no API key and no balance required!

### English Voice Options:
| Voice Name (`TTS_VOICE`) | Gender | Voice Character |
| :--- | :--- | :--- |
| **`en-US-AvaMultilingualNeural`** ⭐ | Female | **Highly recommended!** Modern multilingual voice with rich emotional expression. |
| **`en-US-AndrewMultilingualNeural`** ⭐ | Male | **Highly recommended!** Multilingual with a natural conversational tone. |
| **`en-US-JennyNeural`** | Female | Casual and natural. |
| **`en-US-GuyNeural`** | Male | Clear and steady. |
| **`en-GB-SoniaNeural`** | Female | British accent, warm and polite. |
| **`en-GB-RyanNeural`** | Male | British accent, calm and confident. |
| **`en-AU-NatashaNeural`** | Female | Australian accent, bright and friendly. |
| **`en-IN-NeerjaNeural`** | Female | Indian accent, clear and pleasant. |
| **`en-IN-PrabhatNeural`** | Male | Indian accent, calm and professional. |

### Other Languages:
| Voice Name (`TTS_VOICE`) | Gender | Voice Character |
| :--- | :--- | :--- |
| **`ms-MY-YasminNeural`** | Female | Malay — sweet, soft, friendly, and not stiff at all. |
| **`id-ID-ArdiNeural`** | Male | Indonesian — authoritative, deep, relaxed, and natural. |
| **`id-ID-GadisNeural`** | Female | Indonesian — friendly and clear. |
| **`ms-MY-OsmanNeural`** | Male | Malay — casual, relaxed, and fluent. |
| **`ja-JP-NanamiNeural`** | Female | Japanese — soft. |
| **`hi-IN-SwaraNeural`** | Female | Hindi — warm and expressive. |
| **`es-ES-ElviraNeural`** | Female | Spanish — lively and clear. |
| **`ar-SA-ZariyahNeural`** | Female | Arabic — smooth and calm. |

> ℹ️ The request language is derived automatically from the voice name, so
> `en-US-AvaMultilingualNeural` is sent as `en-US`, `id-ID-ArdiNeural` as `id-ID`, and so on.

### 🎛️ Voice Pitch Settings (`TTS_PITCH`):
- **`-2Hz` / `-3Hz`**: Lowers the pitch so the voice sounds more relaxed, warm, and natural on a phone call.
- **`+0Hz`**: Standard pitch.
- **`+2Hz`**: Slightly higher/brighter pitch.

---

## 👑 4. ElevenLabs Human Voice (`TTS_ENGINE=elevenlabs`)

The most realistic AI voices in the world, with emotional expression, natural speaking tone, and lifelike breathing.

- **Requirement**: Requires `ELEVENLABS_API` (free 10,000 characters/month for new accounts).

| Voice ID (`ELEVEN_VOICE_ID`) | Voice Name | Gender | Voice Character |
| :--- | :--- | :--- | :--- |
| **`21m00Tcm4TlvDq8ikWAM`** ⭐ | **Rachel** | Female | Calm, friendly, and professional (default). |
| **`AZnzlk1XvdvUeBnXmlld`** | **Domi** | Female | Energetic and cheerful. |
| **`EXAVITQu4vr4xnSDxMaL`** | **Bella** | Female | Soft and expressive. |
| **`MF3mGyEYCl7XYWbV9V6O`** | **Elli** | Female | Relaxed and conversational. |
| **`ErXwobaYiN019PkySvjV`** | **Antoni** | Male | Friendly and natural. |
| **`TxGEqnHWrfWFTfGW9XjX`** | **Josh** | Male | Deep, masculine, and natural. |

---

## 🤖 5. OpenAI Human Voice (`TTS_ENGINE=openai`)

Realistic human voices with natural intonation from OpenAI Audio TTS.

- **Requirement**: Requires a paid `OPENAI_API` key (*pay-as-you-go credit balance*).

| Voice Name (`TTS_VOICE`) | Gender | Voice Character |
| :--- | :--- | :--- |
| **`nova`** ⭐ | Female | Friendly, warm, and energetic (default). |
| **`shimmer`** | Female | Clear, soft, and expressive. |
| **`coral`** | Female | Relaxed and natural in tone. |
| **`sage`** | Female | Authoritative and professional. |
| **`alloy`** | Neutral | Balanced and natural. |
| **`echo`** | Male | Soft and calm. |
| **`onyx`** | Male | Deep and authoritative. |
| **`fable`** | Male | Narrative and expressive. |

---

## 🌸 6. Complete Anime VITS TTS Catalog (`TTS_ENGINE=animetts`)

Voice synthesis of anime and game characters (*HuggingFace VITS Synthesizer*).

- **Requirement**: Free, no API key needed.
- **Language (`TTS_LANG`)**: `Mix`, `English`, `日本語`, `简体中文`.

> ⚠️ **Important**: The character names below are exact identifiers required by the
> HuggingFace VITS API. Copy the **entire string verbatim** (including the original
> Chinese/Japanese characters) into the `TTS_CHARACTER` variable in `.env` — do not
> translate or shorten them, otherwise the request will fail.

### 🏇 A. Umamusume Pretty Derby (赛马娘)
Copy the complete character-name string into the `TTS_CHARACTER` variable in `.env`:

1. `特别周 Special Week (Umamusume Pretty Derby)` ⭐ *(Default)*
2. `无声铃鹿 Silence Suzuka (Umamusume Pretty Derby)`
3. `东海帝王 Tokai Teio (Umamusume Pretty Derby)`
4. `丸善斯基 Maruzensky (Umamusume Pretty Derby)`
5. `富士奇迹 Fuji Kiseki (Umamusume Pretty Derby)`
6. `小栗帽 Oguri Cap (Umamusume Pretty Derby)`
7. `黄金船 Gold Ship (Umamusume Pretty Derby)`
8. `伏特加 Vodka (Umamusume Pretty Derby)`
9. `大和赤骥 Daiwa Scarlet (Umamusume Pretty Derby)`
10. `大树快车 Taiki Shuttle (Umamusume Pretty Derby)`
11. `草上飞 Grass Wonder (Umamusume Pretty Derby)`
12. `菱亚马逊 Hishi Amazon (Umamusume Pretty Derby)`
13. `目白麦昆 Mejiro Mcqueen (Umamusume Pretty Derby)`
14. `神鹰 El Condor Pasa (Umamusume Pretty Derby)`
15. `好歌剧 T.M. Opera O (Umamusume Pretty Derby)`
16. `成田白仁 Narita Brian (Umamusume Pretty Derby)`
17. `鲁道夫象征 Symboli Rudolf (Umamusume Pretty Derby)`
18. `气槽 Air Groove (Umamusume Pretty Derby)`
19. `爱丽数码 Agnes Digital (Umamusume Pretty Derby)`
20. `青云天空 Seiun Sky (Umamusume Pretty Derby)`
21. `玉藻十字 Tamamo Cross (Umamusume Pretty Derby)`
22. `美妙姿势 Fine Motion (Umamusume Pretty Derby)`
23. `琵琶晨光 Biwa Hayahide (Umamusume Pretty Derby)`
24. `重炮 Mayano Topgun (Umamusume Pretty Derby)`
25. `曼城茶座 Manhattan Cafe (Umamusume Pretty Derby)`
26. `美普波旁 Mihono Bourbon (Umamusume Pretty Derby)`
27. `目白雷恩 Mejiro Ryan (Umamusume Pretty Derby)`
28. `雪之美人 Yukino Bijin (Umamusume Pretty Derby)`
29. `米浴 Rice Shower (Umamusume Pretty Derby)`
30. `艾尼斯风神 Ines Fujin (Umamusume Pretty Derby)`
31. `爱丽速子 Agnes Tachyon (Umamusume Pretty Derby)`
32. `爱慕织姬 Admire Vega (Umamusume Pretty Derby)`
33. `稻荷一 Inari One (Umamusume Pretty Derby)`
34. `胜利奖券 Winning Ticket (Umamusume Pretty Derby)`
35. `空中神宫 Air Shakur (Umamusume Pretty Derby)`
36. `荣进闪耀 Eishin Flash (Umamusume Pretty Derby)`
37. `真机伶 Curren Chan (Umamusume Pretty Derby)`
38. `川上公主 Kawakami Princess (Umamusume Pretty Derby)`
39. `黄金城市 Gold City (Umamusume Pretty Derby)`
40. `樱花进王 Sakura Bakushin O (Umamusume Pretty Derby)`
41. `采珠 Seeking the Pearl (Umamusume Pretty Derby)`
42. `新光风 Shinko Windy (Umamusume Pretty Derby)`
43. `东商变革 Sweep Tosho (Umamusume Pretty Derby)`
44. `超级小溪 Super Creek (Umamusume Pretty Derby)`
45. `醒目飞鹰 Smart Falcon (Umamusume Pretty Derby)`
46. `荒漠英雄 Zenno Rob Roy (Umamusume Pretty Derby)`
47. `东瀛佐敦 Tosen Jordan (Umamusume Pretty Derby)`
48. `中山庆典 Nakayama Festa (Umamusume Pretty Derby)`
49. `成田大进 Narita Taishin (Umamusume Pretty Derby)`
50. `西野花 Nishino Flower (Umamusume Pretty Derby)`
51. `春乌拉拉 Haru Urara (Umamusume Pretty Derby)`
52. `青竹回忆 Bamboo Memory (Umamusume Pretty Derby)`
53. `待兼福来 Matikane Fukukitaru (Umamusume Pretty Derby)`
54. `名将怒涛 Meisho Doto (Umamusume Pretty Derby)`
55. `目白多伯 Mejiro Dober (Umamusume Pretty Derby)`
56. `优秀素质 Nice Nature (Umamusume Pretty Derby)`
57. `帝王光环 King Halo (Umamusume Pretty Derby)`
58. `待兼诗歌剧 Matikane Tannhauser (Umamusume Pretty Derby)`
59. `生野狄杜斯 Ikuno Dictus (Umamusume Pretty Derby)`
60. `目白善信 Mejiro Palmer (Umamusume Pretty Derby)`
61. `大拓太阳神 Daitaku Helios (Umamusume Pretty Derby)`
62. `双涡轮 Twin Turbo (Umamusume Pretty Derby)`
63. `里见光钻 Satono Diamond (Umamusume Pretty Derby)`
64. `北部玄驹 Kitasan Black (Umamusume Pretty Derby)`
65. `樱花千代王 Sakura Chiyono O (Umamusume Pretty Derby)`
66. `天狼星象征 Sirius Symboli (Umamusume Pretty Derby)`
67. `目白阿尔丹 Mejiro Ardan (Umamusume Pretty Derby)`
68. `八重无敌 Yaeno Muteki (Umamusume Pretty Derby)`
69. `鹤丸刚志 Tsurumaru Tsuyoshi (Umamusume Pretty Derby)`
70. `目白光明 Mejiro Bright (Umamusume Pretty Derby)`
71. `樱花桂冠 Sakura Laurel (Umamusume Pretty Derby)`
72. `成田路 Narita Top Road (Umamusume Pretty Derby)`
73. `也文摄辉 Yamanin Zephyr (Umamusume Pretty Derby)`
74. `真弓快车 Aston Machan (Umamusume Pretty Derby)`
75. `骏川手纲 Hayakawa Tazuna (Umamusume Pretty Derby)`
76. `小林历奇 Kopano Rickey (Umamusume Pretty Derby)`
77. `奇锐骏 Wonder Acute (Umamusume Pretty Derby)`
78. `秋川理事长 President Akikawa (Umamusume Pretty Derby)`

---

### ⚔️ B. Genshin Impact (原神)
1. `派蒙 Paimon (Genshin Impact)`
2. `雷电将军 Raiden Shogun (Genshin Impact)`
3. `钟离 Zhongli (Genshin Impact)`
4. `胡桃 Hu Tao (Genshin Impact)`
5. `甘雨 Ganyu (Genshin Impact)`
6. `纳西妲 Nahida (Genshin Impact)`
7. `八重神子 Yae Miko (Genshin Impact)`
8. `神里绫华 Kamisato Ayaka (Genshin Impact)`
9. `神里绫人 Kamisato Ayato (Genshin Impact)`
10. `枫原万叶 Kaedehara Kazuha (Genshin Impact)`
11. `流浪者 Wanderer (Genshin Impact)`
12. `魈 Xiao (Genshin Impact)`
13. `迪卢克 Diluc (Genshin Impact)`
14. `可莉 Klee (Genshin Impact)`
15. `夜兰 Yelan (Genshin Impact)`
16. `温迪 Venti (Genshin Impact)`
17. `刻晴 Keqing (Genshin Impact)`
18. `琴 Jean (Genshin Impact)`
19. `莫娜 Mona (Genshin Impact)`
20. `妮露 Nilou (Genshin Impact)`
21. `艾尔海森 Alhaitham (Genshin Impact)`
22. `赛诺 Cyno (Genshin Impact)`
23. `提纳里 Tighnari (Genshin Impact)`
24. `柯莱 Collei (Genshin Impact)`
25. `多莉 Dori (Genshin Impact)`
26. `莱依拉 Layla (Genshin Impact)`
27. `坎蒂丝 Candace (Genshin Impact)`
28. `久岐忍 Kuki Shinobu (Genshin Impact)`
29. `鹿野院平藏 Shikanoin Heizou (Genshin Impact)`
30. `云堇 Yun Jin (Genshin Impact)`
31. `申鹤 Shenhe (Genshin Impact)`
32. `宵宫 Yoimiya (Genshin Impact)`
33. `早柚 Sayu (Genshin Impact)`
34. `托马 Thoma (Genshin Impact)`
35. `五郎 Gorou (Genshin Impact)`
36. `荒泷一斗 Arataki Itto (Genshin Impact)`
37. `九条裟罗 Kujou Sara (Genshin Impact)`
38. `珊瑚宫心海 Sangonomiya Kokomi (Genshin Impact)`
39. `埃洛伊 Aloy (Genshin Impact)`
40. `罗莎莉亚 Rosaria (Genshin Impact)`
41. `烟绯 Yanfei (Genshin Impact)`
42. `优菈 Eula (Genshin Impact)`
43. `阿贝多 Albedo (Genshin Impact)`
44. `辛焱 Xinyan (Genshin Impact)`
45. `达达利亚 Tartalia (Genshin Impact)`
46. `迪奥娜 Diona (Genshin Impact)`
47. `重云 Chongyun (Genshin Impact)`
48. `砂糖 Sucrose (Genshin Impact)`
49. `香菱 Xiangling (Genshin Impact)`
50. `行秋 Xingqiu (Genshin Impact)`
51. `诺艾尔 Noelle (Genshin Impact)`
52. `芭芭拉 Barbara (Genshin Impact)`
53. `菲谢尔 Fishl (Genshin Impact)`
54. `奥兹 Oz (Genshin Impact)`
55. `班尼特 Bennett (Genshin Impact)`
56. `雷泽 Razor (Genshin Impact)`
57. `凯亚 Kaeya (Genshin Impact)`
58. `安柏 Amber (Genshin Impact)`
59. `丽莎 Lisa (Genshin Impact)`
60. `七七 Qiqi (Genshin Impact)`
61. `空 Player Male (Genshin Impact)`
62. `荧 Player Female (Genshin Impact)`

---

### 🔮 C. Sanoba Witch (サノバウィッチ)
1. `綾地 寧々 Ayachi Nene (Sanoba Witch)`
2. `因幡 めぐる Inaba Meguru (Sanoba Witch)`
3. `椎葉 紬 Shiiba Tsumugi (Sanoba Witch)`
4. `仮屋 和奏 Kariya Wakama (Sanoba Witch)`
5. `戸隠 憧子 Togakushi Touko (Sanoba Witch)`
