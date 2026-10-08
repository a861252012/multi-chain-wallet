'use strict';
// Source-message catalog: English and Simplified Chinese. Traditional Chinese is the source.
window.FlowMessages = {
  "原交易結果無法確認": ["Original transaction outcome unknown", "原交易结果无法确认"],
  "原交易結果無法確認；該 nonce 已在 finalized 狀態被消耗，目前可建立新交易。請先核對原付款，避免重複支付。": ["The original transaction outcome is unknown. Its nonce has been consumed in finalized state, so a new transaction can now be created. Check the original payment first to avoid paying twice.", "原交易结果无法确认；该 nonce 已在 finalized 状态被消耗，目前可创建新交易。请先核对原付款，避免重复支付。"],
  "原交易結果無法確認；該 nonce 已在 finalized 狀態被消耗。請先核對原付款，避免重複支付。": ["The original transaction outcome is unknown. Its nonce has been consumed in finalized state. Check the original payment first to avoid paying twice.", "原交易结果无法确认；该 nonce 已在 finalized 状态被消耗。请先核对原付款，避免重复支付。"],
  "關閉": ["Close", "关闭"],
  "查詢不會送出交易。若需重試，會沿用原交易，不會建立另一筆付款。": ["Checking does not send a transaction. Retrying uses the original transaction without creating another payment.", "查询不会发送交易。若需重试，会沿用原交易，不会创建另一笔付款。"],
  "無法讀取伺服器回應，請稍後再試。": [
    "Could not read the server response. Please try again later.",
    "无法读取服务器响应，请稍后再试。"
  ],
  "查看訂單": [
    "View order",
    "查看订单"
  ],
  "查看所有交易": [
    "View all transactions",
    "查看所有交易"
  ],
  "這筆訂單屬於其他託管合約，無法在目前合約操作。": [
    "This order belongs to another escrow contract and cannot be managed here.",
    "这笔订单属于其他托管合约，无法在当前合约操作。"
  ],
  "查詢原交易": [
    "Check original transaction",
    "查询原交易"
  ],
  "重試原交易": [
    "Retry original transaction",
    "重试原交易"
  ],
  "交易結果待確認。請先查詢原交易，不要重新建立付款。": [
    "The transaction result is unknown. Check the original transaction before creating another payment.",
    "交易结果待确认。请先查询原交易，不要重新创建付款。"
  ],
  "尚未找到原交易，仍無法確認結果。請稍後再查，或重試原交易。": [
    "The original transaction was not found yet. Check again later or retry the original transaction.",
    "尚未找到原交易，仍无法确认结果。请稍后再查，或重试原交易。"
  ],
  "暫時無法查詢原交易。請稍後再查，不要建立另一筆付款。": [
    "Could not check the original transaction. Try checking later before creating another payment.",
    "暂时无法查询原交易。请稍后再查，不要创建另一笔付款。"
  ],

  "送出時狀態": ["Submission status", "提交时状态"],
  "智慧合約": ["Smart contract", "智能合约"],
  "ETH 存入與取回": ["Deposit and withdraw ETH", "ETH 存入与取回"],
  "測試網": ["Testnet", "测试网"],
  "用測試 ETH 體驗合約操作，從確認費用到查看鏈上結果。": ["Try a smart contract with test ETH, from reviewing fees to checking the on-chain result.", "用测试 ETH 体验合约操作，从确认费用到查看链上结果。"],
  "把測試 ETH 存入智慧合約，再取回目前錢包。每一步都能在鏈上查到交易紀錄。": ["Deposit test ETH into a smart contract, then withdraw it to your current wallet. Each transaction is recorded on-chain.", "把测试 ETH 存入智能合约，再取回当前钱包。每一步都能在链上查到交易记录。"],
  "正在讀取合約狀態…": ["Loading contract status…", "正在读取合约状态…"],
  "目前錢包餘額": ["Current wallet balance", "当前钱包余额"],
  "用於存入與支付手續費": ["Available for deposits and gas fees", "用于存入与支付手续费"],
  "此錢包的合約餘額": ["This wallet’s contract balance", "此钱包的合约余额"],
  "可取回目前錢包的 ETH": ["ETH available to withdraw to this wallet", "可取回当前钱包的 ETH"],
  "選擇操作": ["Choose an action", "选择操作"],
  "存入合約": ["Deposit to contract", "存入合约"],
  "取回錢包": ["Withdraw to wallet", "取回钱包"],
  "目前錢包 → 智慧合約": ["Current wallet → Smart contract", "当前钱包 → 智能合约"],
  "智慧合約 → 目前錢包": ["Smart contract → Current wallet", "智能合约 → 当前钱包"],
  "存入金額（ETH）": ["Deposit amount (ETH)", "存入金额（ETH）"],
  "取回金額（ETH）": ["Withdrawal amount (ETH)", "取回金额（ETH）"],
  "存入後，這筆 ETH 會記在目前錢包地址的合約餘額中。": ["The deposited ETH is credited to your current wallet address in the contract.", "存入后，这笔 ETH 会记在当前钱包地址的合约余额中。"],
  "只能取回此錢包存入的 ETH，款項會回到同一個錢包地址。": ["You can only withdraw ETH deposited by this wallet. It returns to the same wallet address.", "只能取回此钱包存入的 ETH，款项会回到同一个钱包地址。"],
  "存入與取回都需支付手續費（Gas），請在錢包保留一些測試 ETH。": ["Deposits and withdrawals both require gas. Keep some test ETH in your wallet to pay the fees.", "存入与取回都需支付手续费（Gas），请在钱包保留一些测试 ETH。"],
  "下一步會顯示金額與手續費，輸入錢包密碼後才會送出。": ["Review the amount and fees next. The transaction is only sent after you enter your wallet password and confirm.", "下一步会显示金额与手续费，输入钱包密码后才会发送。"],
  "需要測試 ETH？前往領取": ["Need test ETH? Get some here", "需要测试 ETH？前往领取"],
  "這個功能能做什麼？": ["What can I do here?", "这个功能能做什么？"],
  "練習透過智慧合約存入與取回 ETH，查看交易結果與餘額變化。沒有利息，也沒有鎖定期。": ["Practice depositing and withdrawing ETH through a smart contract, then check the results and balances. There is no interest or lock-up period.", "练习通过智能合约存入与取回 ETH，查看交易结果与余额变化。没有利息，也没有锁定期。"],
  "合約依錢包地址記帳；使用同一個錢包的人會共用這筆餘額。": ["The contract tracks balances by wallet address. Anyone using the same wallet shares its balance.", "合约按钱包地址记账；使用同一个钱包的人会共用这笔余额。"],
  "請使用本頁「存入合約」，勿直接轉帳至合約地址。": ["Use “Deposit to contract” on this page. Do not transfer ETH directly to the contract address.", "请使用本页“存入合约”，勿直接转账至合约地址。"],
  "查看操作說明 ↗": ["View instructions ↗", "查看操作说明 ↗"],
  "合約操作": ["Contract operation", "合约操作"],
  "合約操作紀錄": ["Contract activity", "合约操作记录"],
  "尚無合約操作紀錄，完成存入或取回後會顯示在這裡。": ["No contract activity yet. Deposits and withdrawals will appear here.", "尚无合约操作记录，完成存入或取回后会显示在这里。"],
  "收據顯示執行成功；請至智慧合約頁更新餘額。": ["The receipt confirms success. Refresh your balance on the Smart contract page.", "收据显示执行成功；请到智能合约页更新余额。"],
  "正在更新合約餘額…": ["Updating contract balance…", "正在更新合约余额…"],
  "此環境尚未開放合約操作": ["Contract operations are not enabled in this environment", "此环境尚未开放合约操作"],
  "合約尚未設定，目前無法存入或取回 ETH。": ["No contract is configured yet. ETH deposits and withdrawals are unavailable.", "合约尚未设置，目前无法存入或取回 ETH。"],
  "在 Etherscan 查看合約 ↗": ["View contract on Etherscan ↗", "在 Etherscan 查看合约 ↗"],
  "錢包還沒有測試 ETH，請先領取以支付存入或取回的手續費。": ["Your wallet has no test ETH. Get some first to pay the gas fees for deposits or withdrawals.", "钱包还没有测试 ETH，请先领取以支付存入或取回的手续费。"],
  "此錢包尚未存入 ETH。可先試著存入一小筆，再取回錢包。": ["This wallet has not deposited any ETH yet. Try depositing a small amount, then withdrawing it to your wallet.", "此钱包尚未存入 ETH。可先试着存入一小笔，再取回钱包。"],
  "暫時無法讀取餘額": ["Balance temporarily unavailable", "暂时无法读取余额"],
  "請按「更新餘額與紀錄」重試，確認餘額後再操作。": ["Select “Refresh” to retry. Check your balance before proceeding.", "请点击“更新余额与记录”重试，确认余额后再操作。"],
  "正在預估費用…": ["Estimating fees…", "正在估算费用…"],
  "確認存入合約": ["Confirm deposit to contract", "确认存入合约"],
  "確認取回錢包": ["Confirm withdrawal to wallet", "确认取回钱包"],
  "扣款錢包": ["Source wallet", "扣款钱包"],
  "收款錢包": ["Receiving wallet", "收款钱包"],
  "取回說明": ["Withdrawal details", "取回说明"],
  "取回的 ETH 會回到目前錢包；錢包另付 Gas，最多扣除欄位僅為費用上限。": ["The withdrawn ETH returns to your current wallet. Gas is paid separately from your wallet; the maximum debit shown covers gas only.", "取回的 ETH 会回到当前钱包；钱包另付 Gas，最多扣除字段仅为费用上限。"],
  "目前為唯讀展示": ["Read-only demo", "当前为只读展示"],
  "登入後才能查看錢包的合約餘額與操作。": ["Sign in to view your wallet’s contract balance and use contract operations.", "登录后才能查看钱包的合约余额与操作。"],

  "此合約目前僅支援 Ethereum Sepolia 測試網": ["This contract is only supported on Ethereum Sepolia", "此合约目前仅支持 Ethereum Sepolia 测试网"],
  "尚未設定合約地址，暫時無法操作": ["No contract address is configured. Operations are unavailable.", "尚未设置合约地址，暂时无法操作"],
  "合約操作的錢包地址不符，請重新預估": ["The wallet address for this contract operation does not match. Estimate fees again.", "合约操作的钱包地址不符，请重新估算费用"],
  "合約餘額不足，請減少取回金額": ["The contract balance is too low. Reduce the withdrawal amount.", "合约余额不足，请减少取回金额"],
  "合約設定已變更，請重新預估": ["The contract settings have changed. Estimate fees again.", "合约设置已变更，请重新估算费用"],
  "合約地址由伺服器設定，無法在操作時變更": ["The contract address is set by the server and cannot be changed for this operation.", "合约地址由服务器设置，无法在操作时变更"],
  "無法讀取合約回傳的餘額": ["Could not read the balance returned by the contract", "无法读取合约返回的余额"],
  "合約預先檢查未通過，交易尚未送出": ["The contract preflight check failed. No transaction was sent.", "合约预先检查未通过，交易尚未发送"],
  "存入金額必須大於 0": ["The deposit amount must be greater than 0", "存入金额必须大于 0"],
  "取回金額必須大於 0": ["The withdrawal amount must be greater than 0", "取回金额必须大于 0"],
  "僅供測試 · 無利息 · 無鎖定期": ["Test use only · No interest · No lock-up period", "仅供测试 · 无利息 · 无锁定期"],
  "這是共用展示錢包，合約餘額可能隨其他人的操作變動。": ["This is a shared demo wallet. Its contract balance may change when others use it.", "这是共用展示钱包，合约余额可能随其他人的操作变动。"],

  "公開唯讀展示": ["Public read-only demo", "公开只读展示"],
  "公開查詢不需登入。建立、解鎖、簽名及轉帳僅限擁有者。": ["Public queries need no login. Creating, unlocking, signing and sending require owner access.", "公开查询无需登录。创建、解锁、签名及转账仅限所有者。"],
  "擁有者登入": ["Owner sign in", "所有者登录"],
  "查詢地址餘額": ["Check address balance", "查询地址余额"],
  "查核交易": ["Check transaction", "核查交易"],
  "公開唯讀展示 · 未選擇地址": ["Public demo · No address selected", "公开只读展示 · 未选择地址"],
  "錢包交易紀錄僅限擁有者查看；公開交易可至交易查核查詢。": ["Wallet history is private. Look up a public transaction under Transaction check.", "钱包交易记录仅限所有者查看；公开交易可前往交易核查。"],
  "尚未選擇公開觀察地址。": ["No public address selected.", "尚未选择公开观察地址。"],
  "請至地址餘額查詢公開地址；此處不顯示擁有者資產。": ["Look up a public address under Address balance. Owner assets are private.", "请前往地址余额查询公开地址；此处不显示所有者资产。"],
  "此功能僅限擁有者登入後使用。": ["Sign in as the owner to use this feature.", "此功能仅限所有者登录后使用。"],
  "僅限擁有者登入後使用": ["Requires owner sign in", "仅限所有者登录后使用"],

  "目前錢包": ["Current wallet","当前钱包"],
  "切換錢包": ["Switch wallet","切换钱包"],
  "管理錢包": ["Manage wallets","管理钱包"],
  "關閉錢包管理": ["Close wallet manager","关闭钱包管理"],
  "替測試用途取個名字，方便切換。": ["Name wallets by purpose to make switching easier.","按测试用途命名，方便切换。"],
  "新增錢包": ["Add wallet","添加钱包"],
  "顯示已封存": ["Show archived","显示已归档"],
  "封存只會隱藏錢包，保留金鑰與紀錄。可隨時還原，不影響鏈上資產。": ["Archiving hides a wallet and preserves keys and records. Restore it anytime. On-chain assets are unaffected.","归档仅隐藏钱包，保留密钥和记录。可随时恢复，不影响链上资产。"],
  "錢包名稱": ["Wallet name","钱包名称"],
  "例如：轉帳測試、DeFi 練習": ["For example: transfers, DeFi practice","例如：转账测试、DeFi 练习"],
  "下一步：設定錢包": ["Next: set up wallet","下一步：设置钱包"],
  "待完成設定": ["Setup incomplete","待完成设置"],
  "已封存": ["Archived","已归档"],
  "目前使用": ["Current wallet","当前使用"],
  "可切換": ["Available","可切换"],
  "使用錢包": ["Use wallet","使用钱包"],
  "繼續設定": ["Continue setup","继续设置"],
  "更名": ["Rename","重命名"],
  "封存": ["Archive","归档"],
  "還原": ["Restore","恢复"],
  "更改錢包名稱": ["Rename wallet","重命名钱包"],
  "封存錢包": ["Archive wallet","归档钱包"],
  "先命名，下一步建立新錢包或匯入現有錢包。每個錢包分開保存金鑰與紀錄。": ["Choose a name, then create or import a wallet. Each wallet stores its keys and records separately.","先命名，下一步创建或导入钱包。每个钱包分别保存密钥和记录。"],
  "封存後會從切換清單隱藏，金鑰與紀錄仍會保留。這不會刪除或轉移鏈上資產。": ["This hides the wallet from the switcher and keeps keys and records. It does not delete or transfer on-chain assets.","归档后从切换列表隐藏，保留密钥和记录。不会删除或转移链上资产。"],
  "還原後會重新出現在錢包切換清單。": ["The wallet will reappear in the wallet switcher.","恢复后重新出现在钱包切换列表。"],
  "名稱只用於本機辨識，不會改變錢包地址。": ["Names are local labels and do not change wallet addresses.","名称仅用于本机识别，不会改变钱包地址。"],
  "已儲存": ["Saved","已保存"],
  "請至少保留一個未封存的錢包": ["Keep at least one wallet unarchived","请至少保留一个未归档的钱包"],
  "錢包名稱請填寫 1 至 40 個字，且不可含控制字元": ["Use 1–40 characters without control characters","钱包名称需为 1 至 40 个字符，不可包含控制字符"],
  "最小單位數量（整數）": ["Smallest-unit amount (integer)", "最小单位数量（整数）"],
  "代幣精度": ["Token decimals", "代币精度"],
  "自訂代幣直接簽署這個最小單位整數；請依可信來源核對合約與精度。手續費以 ${nativeSymbol} 支付。": ["This exact smallest-unit integer will be signed for the custom token. Verify the contract and decimals with a trusted source. Fees are paid in ${nativeSymbol}.", "自定义代币会直接签署这个最小单位整数；请通过可信来源核对合约与精度。手续费以 ${nativeSymbol} 支付。"],
  "自訂代幣必須輸入最小單位整數": ["Enter the custom-token amount as a smallest-unit integer", "自定义代币必须输入最小单位整数"],
  "自訂代幣最小單位只能是十進位整數": ["The custom-token smallest-unit amount must be a decimal integer", "自定义代币最小单位只能是十进制整数"],
  "自訂代幣最小單位超出 uint256 範圍": ["The custom-token smallest-unit amount exceeds the uint256 range", "自定义代币最小单位超出 uint256 范围"],
  "RPC 回傳的代幣資料與內建登錄不符": ["RPC token metadata does not match the built-in registry", "RPC 返回的代币资料与内置登记不符"],
  "請求過於頻繁，請稍後再試。": ["Too many requests. Please try again later.", "请求过于频繁，请稍后重试。"],
  "服務忙碌，這次請求尚未執行，請稍後再試。": ["Service busy. This request was not executed. Please try again later.", "服务繁忙，本次请求尚未执行，请稍后重试。"],
  "地址、收據與交易詳情": ["Address, receipt and details", "地址、收据与交易详情"],
  "交易池與滑價設定": ["Pool and slippage settings", "交易池与滑点设置"],
  "查看餘額，或選擇下一步操作。": ["Check your balance or choose what to do next.", "查看余额，或选择下一步操作。"],
  "填入收款地址與數量，下一步核對費用。": ["Enter a recipient and amount, then review the fees.", "填入收款地址与数量，下一步核对费用。"],
  "分享地址，請確認對方使用相同測試網路。": ["Share your address. Use the same test network.", "分享地址，请确认对方使用相同测试网络。"],
  "選擇支付資產與數量，逐步完成鏈上兌換。": ["Choose an asset and amount, then follow the swap steps.", "选择支付资产与数量，逐步完成链上兑换。"],
  "補充測試餘額，開始體驗收付款。": ["Get test tokens to try sending and receiving.", "补充测试余额，开始体验收付款。"],
  "追蹤送出的交易，查看結果與處理進度。": ["Track your outgoing transactions and their latest status.", "追踪发出的交易，查看结果与处理进度。"],
  "核對收付款與手續費，匯出需要的紀錄。": ["Review transfers and fees, and export your records.", "核对收付款与手续费，导出需要的记录。"],
  "管理加密備份與錢包密碼。": ["Manage encrypted backups and your wallet password.", "管理加密备份与钱包密码。"],
  "儲存常用收款地址，下次轉帳直接選用。": ["Save recipients for your next transfer.", "保存常用收款地址，下次转账直接选用。"],

  "Multi-Chain Wallet · 錢包": [
    "Multi-Chain Wallet · Wallet",
    "Multi-Chain Wallet · 钱包"
  ],
  "跳至主要內容": [
    "Skip to main content",
    "跳至主要内容"
  ],
  "測試鏈工作區": [
    "Testnet workspace",
    "测试链工作区"
  ],
  "錢包": [
    "Wallet",
    "钱包"
  ],
  "總覽": [
    "Overview",
    "总览"
  ],
  "發送資產": [
    "Send",
    "发送资产"
  ],
  "收款": [
    "Receive",
    "收款"
  ],
  "資產兌換": [
    "Exchange",
    "资产兑换"
  ],
  "領取測試幣": [
    "Get test tokens",
    "领取测试币"
  ],
  "管理": [
    "Manage",
    "管理"
  ],
  "交易紀錄": [
    "Transactions",
    "交易记录"
  ],
  "收支流水": [
    "Activity",
    "收支流水"
  ],
  "地址簿": [
    "Address book",
    "地址簿"
  ],
  "設定與備份": [
    "Settings & backup",
    "设置与备份"
  ],
  "進階工具": [
    "Advanced tools",
    "高级工具"
  ],
  "唯讀觀察": [
    "Watch address",
    "唯读观察"
  ],
  "地址餘額": [
    "Balance lookup",
    "地址余额"
  ],
  "交易查核": [
    "Transaction lookup",
    "交易查核"
  ],
  "交易診斷": [
    "Diagnostics",
    "交易诊断"
  ],
  "操作指南": [
    "Getting started",
    "操作指南"
  ],
  "僅供測試鏈使用": [
    "Testnets only",
    "仅供测试链使用"
  ],
  "作品與驗證 ↗": [
    "Portfolio & evidence ↗",
    "作品与验证 ↗"
  ],
  "測試網路": [
    "Network",
    "测试网络"
  ],
  "暗黑模式": [
    "Dark mode",
    "暗黑模式"
  ],
  "管理測試資產，從這裡開始。": [
    "Your test assets, in one place.",
    "管理测试资产，从这里开始。"
  ],
  "目前帳戶": [
    "Account",
    "目前账户"
  ],
  "新增獨立帳戶": [
    "Add account",
    "添加独立账户"
  ],
  "正在載入本機錢包…": [
    "Loading your local wallet…",
    "正在加载本机钱包…"
  ],
  "建立錢包，": [
    "Create a wallet.",
    "创建钱包，"
  ],
  "開始體驗收付款。": [
    "Start sending and receiving.",
    "开始体验收付款。"
  ],
  "設定密碼、備份助記詞，就能開始使用。已有測試錢包，也可以用助記詞還原。": [
    "Set a password and back up your recovery phrase. You can also restore an existing test wallet.",
    "设置密码、备份助记词，就能开始使用。已有测试钱包，也可以用助记词还原。"
  ],
  "助記詞用來還原錢包，請離線備份": [
    "Back up your recovery phrase offline to restore your wallet",
    "助记词用来还原钱包，请脱机备份"
  ],
  "助記詞只在建立時顯示一次": [
    "The recovery phrase is shown only once",
    "助记词只在创建时显示一次"
  ],
  "密碼用來保護本機錢包與確認交易": [
    "Your password protects the local wallet and confirms transactions",
    "密码用来保护本机钱包与确认交易"
  ],
  "僅供測試鏈使用。請勿匯入持有真實資產的助記詞。": [
    "Testnets only. Never import a recovery phrase that controls real assets.",
    "仅供测试链使用。请勿导入持有真实资产的助记词。"
  ],
  "進階資訊：帳戶與還原規格": [
    "Advanced: accounts and recovery",
    "高级信息：账户与还原规格"
  ],
  "錢包使用 Go 在本機產生帳戶，並以密碼加密私鑰。": [
    "Go generates your account locally and encrypts the private key with your password.",
    "钱包使用 Go 在本机产生账户，并以密码加密私钥。"
  ],
  "新錢包使用 12 個英文單字的 BIP-39 助記詞，依 BIP-44 路徑": [
    "New wallets use a 12-word English BIP-39 phrase and the BIP-44 path",
    "新钱包使用 12 个英文单字的 BIP-39 助记词，依 BIP-44 路径"
  ],
  "取得一個 Ethereum 地址。": [
    "to derive an Ethereum address.",
    "取得一个 Ethereum 地址。"
  ],
  "還原使用相同路徑；其他路徑的地址不會自動載入。目前不支援額外密語（BIP-39 passphrase）。": [
    "Recovery uses the same path. Other paths are not loaded automatically. Additional BIP-39 passphrases are not supported.",
    "还原使用相同路径；其他路径的地址不会自动加载。目前不支持额外密语（BIP-39 passphrase）。"
  ],
  "建立新錢包": [
    "Create wallet",
    "创建新钱包"
  ],
  "還原錢包": [
    "Restore wallet",
    "还原钱包"
  ],
  "英文助記詞": [
    "English recovery phrase",
    "英文助记词"
  ],
  "依原順序輸入，以空白分隔。只還原預設路徑的第一個 Ethereum 地址；若原錢包另設助記詞額外密語，目前無法還原。": [
    "Enter words in their original order, separated by spaces. Only the first Ethereum address on the default path is restored. Additional BIP-39 passphrases are not supported.",
    "依原顺序输入，以空白分隔。只还原缺省路径的第一个 Ethereum 地址；若原钱包另设助记词额外密语，目前无法还原。"
  ],
  "錢包密碼": [
    "Wallet password",
    "钱包密码"
  ],
  "至少 12 字元，用於確認交易與保護這台電腦上的錢包。請另外備份助記詞，密碼無法取代它。": [
    "Use at least 12 characters to protect this local wallet and confirm transactions. Back up your recovery phrase separately; the password cannot replace it.",
    "至少 12 字符，用于确认交易与保护这台电脑上的钱包。请另外备份助记词，密码无法取代它。"
  ],
  "再次輸入密碼": [
    "Confirm password",
    "再次输入密码"
  ],
  "建立錢包": [
    "Create wallet",
    "创建钱包"
  ],
  "從加密備份還原": [
    "Restore encrypted backup",
    "从加密备份还原"
  ],
  "備份原密碼": [
    "Original backup password",
    "备份原密码"
  ],
  "新密碼（12–128 字元）": [
    "New password (12–128 characters)",
    "新密码（12–128 字符）"
  ],
  "再次輸入新密碼": [
    "Confirm new password",
    "再次输入新密码"
  ],
  "匯入備份": [
    "Import backup",
    "导入备份"
  ],
  "先備份，再開始。": [
    "Back up before you begin.",
    "先备份，再开始。"
  ],
  "請將這些單字依序離線抄下。擁有助記詞的人可以控制錢包；請勿傳給任何人或貼到 AI 對話。": [
    "Write these words down in order and keep them offline. Anyone with this phrase controls the wallet. Never share it or paste it into an AI chat.",
    "请将这些单字依序脱机抄下。拥有助记词的人可以控制钱包；请勿传给任何人或贴到 AI 对话。"
  ],
  "我已依序離線備份，知道關閉後不會再次顯示。": [
    "I have backed up the words in order and understand they will not be shown again.",
    "我已依序脱机备份，知道关闭后不会再次显示。"
  ],
  "備份完成，進入錢包": [
    "Backup complete · Open wallet",
    "备份完成，进入钱包"
  ],
  "可用餘額": [
    "Available balance",
    "可用余额"
  ],
  "等待鏈上餘額": [
    "Waiting for on-chain balance",
    "等待链上余额"
  ],
  "更新餘額與紀錄": [
    "Refresh",
    "更新余额与记录"
  ],
  "收款地址": [
    "Receiving address",
    "收款地址"
  ],
  "複製地址": [
    "Copy address",
    "拷贝地址"
  ],
  "外部 ETH 水龍頭 ↗": [
    "External ETH faucet ↗",
    "外部 ETH 水龙头 ↗"
  ],
  "在區塊瀏覽器查看 ↗": [
    "View in explorer ↗",
    "在区块浏览器查看 ↗"
  ],
  "僅接收 Ethereum Sepolia 的 ETH 與代幣。": [
    "Only receive Ethereum Sepolia ETH and tokens.",
    "仅接收 Ethereum Sepolia 的 ETH 与代币。"
  ],
  "複製地址後，請使用相同測試網路轉帳。": [
    "Use the same test network when transferring to this address.",
    "拷贝地址后，请使用相同测试网络转账。"
  ],
  "轉帳到另一個地址": [
    "Transfer to another address",
    "转账到另一个地址"
  ],
  "複製收款地址": [
    "Copy receiving address",
    "复制收款地址"
  ],
  "交換測試 ETH 與 USDC": [
    "Swap test ETH and USDC",
    "交换测试 ETH 与 USDC"
  ],
  "補充測試用餘額": [
    "Top up your test balance",
    "补充测试用余额"
  ],
  "下載加密備份": [
    "Download encrypted backup",
    "下载加密备份"
  ],
  "輸入錢包密碼": [
    "Enter wallet password",
    "输入钱包密码"
  ],
  "下載加密備份檔": [
    "Download backup file",
    "下载加密备份档"
  ],
  "每種幣每小時可領一次。領取後自動更新餘額。": [
    "One claim per token per hour. Your balance refreshes after the transfer.",
    "每种币每小时可领一次。领取后自动更新余额。"
  ],
  "領取 0.01 測試 USDC": [
    "Get 0.01 test USDC",
    "领取 0.01 测试 USDC"
  ],
  "領取後會顯示交易連結，並更新餘額。": [
    "The transaction link and updated balance will appear here.",
    "领取后会显示交易链接，并更新余额。"
  ],
  "從測試幣開始，完成第一筆上鏈。": [
    "Your first on-chain transaction.",
    "从测试币开始，完成第一笔上链。"
  ],
  "先按上方「領取測試幣」，收到 ETH 後即可支付手續費。再領取測試 USDC，體驗代幣轉帳與兌換；庫存不足時可改用以下外部水龍頭。": [
    "Get test tokens first. ETH pays transaction fees; test USDC lets you try transfers and swaps. Use an external faucet if local inventory is empty.",
    "先按上方「领取测试币」，收到 ETH 后即可支付手续费。再领取测试 USDC，体验代币转账与兑换；库存不足时可改用以下外部水龙头。"
  ],
  "領取 Sepolia ETH ↗": [
    "Get Sepolia ETH ↗",
    "领取 Sepolia ETH ↗"
  ],
  "領取 Circle 測試 USDC ↗": [
    "Get Circle test USDC ↗",
    "领取 Circle 测试 USDC ↗"
  ],
  "其他 ETH 水龍頭 ↗": [
    "Other ETH faucets ↗",
    "其他 ETH 水龙头 ↗"
  ],
  "外部服務可能要求登入、驗證或限制領取次數。不需要購買測試幣，也不要提供助記詞或私鑰。": [
    "External services may require sign-in, verification or impose limits. Do not buy test tokens or share your recovery phrase or private key.",
    "外部服务可能要求登录、验证或限制领取次数。不需要购买测试币，也不要提供助记词或私钥。"
  ],
  "我已領取，查詢餘額": [
    "I claimed tokens · Check balance",
    "我已领取，查找余额"
  ],
  "等待鏈上餘額。": [
    "Waiting for on-chain balance.",
    "等待链上余额。"
  ],
  "試一筆小額交易": [
    "Try a small transfer",
    "试一笔小额交易"
  ],
  "先把 0.000001 ETH 轉回自己的地址，體驗轉帳與查看結果。本金回到原地址，另付測試 ETH 手續費。": [
    "Send 0.000001 ETH to your own address to try a transaction. The principal returns to you; test ETH fees are still charged.",
    "先把 0.000001 ETH 转回自己的地址，体验转账与查看结果。本金回到原地址，另付测试 ETH 手续费。"
  ],
  "填入小額自轉": [
    "Fill a small self-transfer",
    "填入小额自转"
  ],
  "接著可將 0.000001 ETH 包裝成 WETH，再到兌換區授權與交換測試 USDC。": [
    "You can then wrap 0.000001 ETH into WETH and approve and swap test USDC in Exchange.",
    "接着可将 0.000001 ETH 包装成 WETH，再到兑换区授权与交换测试 USDC。"
  ],
  "填入 WETH 包裝": [
    "Fill a WETH wrap",
    "填入 WETH 包装"
  ],
  "按鈕只填入表單。請核對報價與最高費用，再輸入密碼簽署。": [
    "These buttons only fill the form. Review the quote and maximum fees before signing with your password.",
    "按钮只填入表单。请核对报价与最高费用，再输入密码签署。"
  ],
  "用收據確認結果": [
    "Verify the receipt",
    "用收据确认结果"
  ],
  "送出後更新交易紀錄，確認顯示「鏈上執行成功」，並開啟 Etherscan 核對交易雜湊、網路與收據。": [
    "Refresh transactions after sending. Check for execution success, then verify the hash, network and receipt in the explorer.",
    "送出后更新交易记录，确认显示「链上运行成功」，并打开 Etherscan 核对交易哈希、网络与收据。"
  ],
  "查看交易紀錄 →": [
    "View transactions →",
    "查看交易记录 →"
  ],
  "核對收支與匯出 CSV →": [
    "Review activity & export CSV →",
    "核对收支与导出 CSV →"
  ],
  "顯示「已廣播」仍須等待區塊收錄。若結果待確認，先更新紀錄；需要重試時使用「重新廣播原交易」。": [
    "Broadcast transactions still need block inclusion. If the result is unknown, refresh first. Use Rebroadcast original transaction when retrying.",
    "显示「已广播」仍须等待区块收录。若结果待确认，先更新记录；需要重试时使用「重新广播原交易」。"
  ],
  "資產": [
    "Asset",
    "资产"
  ],
  "操作": [
    "Action",
    "操作"
  ],
  "轉帳給收款人": [
    "Transfer to recipient",
    "转账给收款人"
  ],
  "授權地址使用代幣": [
    "Approve token spending",
    "授权地址使用代币"
  ],
  "授權會允許被授權地址花費指定數量。輸入 0 可撤銷；修改既有額度前須先歸零。": [
    "Approval allows the address to spend the specified amount. Enter 0 to revoke. Clear an existing allowance before changing it.",
    "授权会允许被授权地址花费指定数量。输入 0 可撤销；修改既有额度前须先归零。"
  ],
  "從地址簿填入": [
    "From address book",
    "从地址簿填入"
  ],
  "選擇收款人": [
    "Choose recipient",
    "选择收款人"
  ],
  "金額": [
    "Amount",
    "金额"
  ],
  "手續費另以 {{.Native}} 支付。": [
    "Fees are paid separately in {{.Native}}.",
    "手续费另以 {{.Native}} 支付。"
  ],
  "預估費用並核對": [
    "Review transfer",
    "预估费用并核对"
  ],
  "我的代幣": [
    "My tokens",
    "我的代币"
  ],
  "加入代幣": [
    "Add token",
    "加入代币"
  ],
  "目前網路的代幣合約": [
    "Token contract on this network",
    "目前网络的代币合约"
  ],
  "查詢並加入": [
    "Find & add",
    "查找并加入"
  ],
  "尚未加入代幣。": [
    "No tokens added yet.",
    "尚未加入代币。"
  ],
  "ETH 與 WETH 可 1:1 包裝／解包。WETH 與測試 USDC 使用 Uniswap V3 的鏈上交易池；測試網價格不代表美元價值。": [
    "Wrap and unwrap ETH and WETH at 1:1. WETH and test USDC swaps use Uniswap V3 pools. Testnet prices do not represent dollar values.",
    "ETH 与 WETH 可 1:1 包装／解包。WETH 与测试 USDC 使用 Uniswap V3 的链上交易池；测试网价格不代表美元价值。"
  ],
  "兌換方向": [
    "Swap direction",
    "兑换方向"
  ],
  "ETH → 測試 USDC（引導流程）": [
    "ETH → Test USDC (guided)",
    "ETH → 测试 USDC（引导流程）"
  ],
  "測試 USDC → ETH（引導流程）": [
    "Test USDC → ETH (guided)",
    "测试 USDC → ETH（引导流程）"
  ],
  "ETH → WETH（包裝）": [
    "ETH → WETH (wrap)",
    "ETH → WETH（包装）"
  ],
  "WETH → ETH（解包）": [
    "WETH → ETH (unwrap)",
    "WETH → ETH（解包）"
  ],
  "WETH → 測試 USDC": [
    "WETH → Test USDC",
    "WETH → 测试 USDC"
  ],
  "測試 USDC → WETH": [
    "Test USDC → WETH",
    "测试 USDC → WETH"
  ],
  "支付數量": [
    "You pay",
    "支付数量"
  ],
  "結束引導（保留已完成交易）": [
    "End guide (keep completed transactions)",
    "结束引导（保留已完成交易）"
  ],
  "1. 查詢餘額與授權 → 2. 必要時授權並等待成功 → 3. 取得報價、核對最低收到數量 → 4. 簽署兌換。": [
    "1. Check balance and allowance → 2. Approve if needed and wait → 3. Review the quote and minimum received → 4. Sign the swap.",
    "1. 查找余额与授权 → 2. 必要时授权并等待成功 → 3. 取得报价、核对最低收到数量 → 4. 签署兑换。"
  ],
  "查詢餘額與授權": [
    "Check balance & allowance",
    "查找余额与授权"
  ],
  "選好支付數量後，可先查詢是否需要授權。": [
    "Enter an amount to check whether approval is needed.",
    "选好支付数量后，可先查找是否需要授权。"
  ],
  "交易池費率": [
    "Pool fee",
    "交易池费率"
  ],
  "自動比較可用交易池": [
    "Compare available pools automatically",
    "自动比较可用交易池"
  ],
  "比較交易池報價": [
    "Compare pool quotes",
    "比较交易池报价"
  ],
  "允許滑價": [
    "Slippage tolerance",
    "允许滑价"
  ],
  "先授權本次支付數量，等待交易成功後再報價。已有非零額度時，可先撤銷。": [
    "Approve only the amount you plan to spend, then wait for success before requesting a quote. Revoke an existing allowance first if needed.",
    "先授权本次支付数量，等待交易成功后再报价。已有非零额度时，可先撤销。"
  ],
  "授權本次數量": [
    "Approve this amount",
    "授权本次数量"
  ],
  "撤銷兌換合約授權": [
    "Revoke swap allowance",
    "撤销兑换合约授权"
  ],
  "取得鏈上報價並核對": [
    "Get quote & review",
    "取得链上报价并核对"
  ],
  "查看收付款與手續費，或匯出本頁紀錄。": [
    "Review transfers and fees, or export the current page.",
    "查看收付款与手续费，或导出本页记录。"
  ],
  "更新流水": [
    "Refresh activity",
    "更新流水"
  ],
  "匯出本頁 CSV": [
    "Export page as CSV",
    "导出本页 CSV"
  ],
  "同步與匯入設定": [
    "Sync & import settings",
    "同步与导入设置"
  ],
  "自動同步起始區塊（留空續掃；首次從最近已確定區塊開始）": [
    "Sync start block (leave blank to resume; first run starts at a recent finalized block)",
    "自动同步起始区块（留空续扫；首次从最近已确定区块开始）"
  ],
  "開始／繼續同步與發現代幣": [
    "Start / resume sync & token discovery",
    "开始／继续同步与发现代币"
  ],
  "暫停同步": [
    "Pause sync",
    "暂停同步"
  ],
  "同步進度載入中…": [
    "Loading sync progress…",
    "同步进度加载中…"
  ],
  "同步範圍以前端顯示的起始區塊為準，只掃描節點回報已達終局性的區塊；節點不支援或失敗時會保留進度。": [
    "Scanning starts at the displayed block and only includes blocks the node reports as finalized. Progress is retained if the node fails or does not support finality.",
    "同步范围以前端显示的起始区块为准，只扫描节点回报已达终局性的区块；节点不支持或失败时会保留进度。"
  ],
  "起始區塊（留空查最近 20 個區塊）": [
    "Start block (blank = latest 20 blocks)",
    "起始区块（留空查最近 20 个区块）"
  ],
  "同步鏈上收支": [
    "Sync on-chain activity",
    "同步链上收支"
  ],
  "匯入交易雜湊": [
    "Import transaction hash",
    "导入交易哈希"
  ],
  "查核並匯入": [
    "Verify & import",
    "查核并导入"
  ],
  "建立錢包後即可開始記錄。": [
    "Create a wallet to start recording activity.",
    "创建钱包后即可开始记录。"
  ],
  "上一頁": [
    "Previous",
    "上一页"
  ],
  "下一頁": [
    "Next",
    "下一页"
  ],
  "合計只涵蓋本頁已查核交易，不是錢包餘額或完整帳務。部分收支可能未列入，請一併核對餘額。": [
    "Totals cover verified transactions on this page, not your full balance or accounting history. Some movements may be missing; also check your balance.",
    "合计只涵盖本页已查核交易，不是钱包余额或完整帐务。部分收支可能未列入，请一并核对余额。"
  ],
  "收支紀錄的範圍": [
    "Activity coverage",
    "收支记录的范围"
  ],
  "未收錄的區塊、一般合約內部 ETH 轉帳與非標準代幣可能未列入；WETH 解包另依 Withdrawal 事件辨識。ERC-20 收支以事件為依據。": [
    "Unindexed blocks, general internal ETH transfers and nonstandard tokens may be missing. WETH unwraps use Withdrawal events; ERC-20 movements use transfer events.",
    "未收录的区块、一般合约内部 ETH 转账与非标准代币可能未列入；WETH 解包另依 Withdrawal 事件辨识。ERC-20 收支以事件为依据。"
  ],
  "安全性：更改錢包密碼": [
    "Security: change wallet password",
    "安全性：更改钱包密码"
  ],
  "更改密碼": [
    "Change password",
    "更改密码"
  ],
  "更改後請重新下載加密備份，舊備份仍使用舊密碼。": [
    "Download a new encrypted backup after changing your password. Older backups still use the old password.",
    "更改后请重新下载加密备份，旧备份仍使用旧密码。"
  ],
  "目前密碼": [
    "Current password",
    "目前密码"
  ],
  "更新密碼": [
    "Update password",
    "更新密码"
  ],
  "查看全部交易 →": [
    "View all transactions →",
    "查看全部交易 →"
  ],
  "由此錢包送出的交易與最新查核狀態。": [
    "Transactions sent from this wallet and their latest verified status.",
    "由此钱包送出的交易与最新查核状态。"
  ],
  "搜尋交易（地址、雜湊、資產或操作）": [
    "Search by address, hash, asset or action",
    "搜索交易（地址、哈希、资产或操作）"
  ],
  "尚無交易。": [
    "No transactions yet.",
    "尚无交易。"
  ],
  "確認這筆交易": [
    "Review transaction",
    "确认这笔交易"
  ],
  "取消": [
    "Cancel",
    "取消"
  ],
  "這筆操作會授權指定地址花費你的代幣，並不是轉帳。請確認被授權地址值得信任。": [
    "This approves another address to spend your tokens; it is not a transfer. Verify that you trust the spender.",
    "这笔操作会授权指定地址花费你的代币，并不是转账。请确认被授权地址值得信任。"
  ],
  "最高費用是上限，實際手續費依交易執行結果而定。報價逾時需重新預估。": [
    "The maximum fee is a cap. Actual fees depend on execution. Request a new quote if this one expires.",
    "最高费用是上限，实际手续费依交易运行结果而定。报价逾时需重新预估。"
  ],
  "簽署並送出至 Sepolia": [
    "Sign & send to Sepolia",
    "签署并送出至 Sepolia"
  ],
  "只需公開地址，即可查詢資產與鏈上收據。這裡不會建立錢包或發送交易。": [
    "View assets and receipts using a public address. This does not create a wallet or send transactions.",
    "只需公开地址，即可查找资产与链上收据。这里不会创建钱包或发送交易。"
  ],
  "觀察地址": [
    "Watch address",
    "观察地址"
  ],
  "代幣合約（選填）": [
    "Token contract (optional)",
    "代币合约（选填）"
  ],
  "查詢資產": [
    "Check assets",
    "查找资产"
  ],
  "儲存觀察地址": [
    "Save watch address",
    "保存观察地址"
  ],
  "查看哪個區塊（留白為最新 finalized）": [
    "Block to inspect (blank = latest finalized)",
    "查看哪个区块（留白为最新 finalized）"
  ],
  "查詢此區塊的相關交易": [
    "Find transactions in this block",
    "查找此区块的相关交易"
  ],
  "或查核指定交易雜湊": [
    "Or inspect a transaction hash",
    "或查核指定交易哈希"
  ],
  "核對收支收據": [
    "Verify activity receipt",
    "核对收支收据"
  ],
  "標籤與觀察地址只儲存在此瀏覽器，依測試網路分開。請核對地址後再轉帳。": [
    "Labels and watch addresses are stored in this browser and separated by test network. Verify the address before sending.",
    "标签与观察地址只保存在此浏览器，依测试网络分开。请核对地址后再转账。"
  ],
  "名稱": [
    "Name",
    "名称"
  ],
  "完整地址": [
    "Full address",
    "完整地址"
  ],
  "儲存地址": [
    "Save address",
    "保存地址"
  ],
  "網路與交易診斷": [
    "Network & transaction diagnostics",
    "网络与交易诊断"
  ],
  "更新診斷": [
    "Refresh diagnostics",
    "更新诊断"
  ],
  "計數從本次程序啟動累計。延遲涵蓋網路檢查與 HTTP 回應標頭；傳輸成功不等於交易成功。同步範圍請見收支流水。": [
    "Counters start when this process launches. Latency includes network checks and HTTP response headers; transport success does not prove execution success. See Activity for scan coverage.",
    "计数从本次进程启动累计。延迟涵盖网络检查与 HTTP 回应标头；传输成功不等于交易成功。同步范围请见收支流水。"
  ],
  "交易雜湊": [
    "Transaction hash",
    "交易哈希"
  ],
  "查看交易生命週期": [
    "View transaction lifecycle",
    "查看交易生命周期"
  ],
  "網路概況": [
    "Network overview",
    "网络概况"
  ],
  "更新網路": [
    "Refresh network",
    "更新网络"
  ],
  "目前網路": [
    "Current network",
    "目前网络"
  ],
  "查詢中…": [
    "Loading…",
    "查找中…"
  ],
  "最新區塊": [
    "Latest block",
    "最新区块"
  ],
  "等待鏈上資料": [
    "Waiting for on-chain data",
    "等待链上数据"
  ],
  "RPC 連線": [
    "RPC connection",
    "RPC 连接"
  ],
  "尚未取得資料": [
    "No data yet",
    "尚未取得数据"
  ],
  "查詢目前網路任意地址的 {{.Native}} 餘額。": [
    "Check the {{.Native}} balance of any address on this network.",
    "查找目前网络任意地址的 {{.Native}} 余额。"
  ],
  "錢包或合約地址": [
    "Wallet or contract address",
    "钱包或合约地址"
  ],
  "必填": [
    "Required",
    "必填"
  ],
  "貼上目前測試網路的完整收款地址。": [
    "Paste the complete receiving address for this test network.",
    "粘贴目前测试网络的完整收款地址。"
  ],
  "查詢餘額": [
    "Check balance",
    "查找余额"
  ],
  "你的第一筆查核，從這裡開始": [
    "Look up an address",
    "你的第一笔查核，从这里开始"
  ],
  "貼上公開地址，即可查看 {{.Native}} 餘額。": [
    "Paste a public address to view its {{.Native}} balance.",
    "粘贴公开地址，即可查看 {{.Native}} 余额。"
  ],
  "從交易收據，確認執行結果與實際費用。": [
    "Check execution and actual fees from the receipt.",
    "从交易收据，确认运行结果与实际费用。"
  ],
  "請從交易紀錄複製交易雜湊，這與收款地址不同。": [
    "Copy the transaction hash from your history. It is different from a receiving address.",
    "请从交易记录拷贝交易哈希，这与收款地址不同。"
  ],
  "查詢交易": [
    "Check transaction",
    "查找交易"
  ],
  "不只看成功，也看確認進度": [
    "Track execution and confirmations",
    "不只看成功，也看确认进度"
  ],
  "查詢交易結果、確認進度與實際手續費。": [
    "Look up execution status, confirmations and actual fees.",
    "查找交易结果、确认进度与实际手续费。"
  ],
  "請啟用 JavaScript，才能查詢網路、餘額與交易。": [
    "Enable JavaScript to use network, balance and transaction queries.",
    "请激活 JavaScript，才能查找网络、余额与交易。"
  ],
  "主要導覽": [
    "Main navigation",
    "主要导览"
  ],
  "關閉選單": [
    "Close menu",
    "关闭菜单"
  ],
  "開啟選單": [
    "Open menu",
    "打开菜单"
  ],
  "錢包設定方式": [
    "Wallet setup method",
    "钱包设置方式"
  ],
  "最近 20 個區塊": [
    "Latest 20 blocks",
    "最近 20 个区块"
  ],
  "輸入關鍵字": [
    "Search transactions",
    "输入关键字"
  ],
  "例如：我的第二個測試錢包": [
    "For example: My second test wallet",
    "例如：我的第二个测试钱包"
  ],
  "網路狀態": [
    "Network status",
    "网络状态"
  ],
  "0x 開頭的 42 字元地址": [
    "42-character address starting with 0x",
    "0x 开头的 42 字符地址"
  ],
  "0x 開頭的 66 字元雜湊": [
    "66-character hash starting with 0x",
    "0x 开头的 66 字符哈希"
  ],
  "Multi-Chain Wallet · 作品與驗證": [
    "Multi-Chain Wallet · Portfolio & evidence",
    "Multi-Chain Wallet · 作品与验证"
  ],
  "← 開啟錢包": [
    "← Open wallet",
    "← 打开钱包"
  ],
  "測試網轉帳、兌換": [
    "Testnet transfers, swaps",
    "测试网转账、兑换"
  ],
  "與交易復原。": [
    "and transaction recovery.",
    "与交易恢复。"
  ],
  "用 Go 實作多鏈測試網錢包，練習簽署交易、預估手續費與處理失敗。": [
    "A Go multichain testnet wallet for practicing transaction signing, fee estimation and failure handling.",
    "用 Go 实现多链测试网钱包，练习签署交易、预估手续费与处理失败。"
  ],
  "先操作，再深入": [
    "Try it, then explore the design",
    "先操作，再深入"
  ],
  "01 · 多鏈錢包": [
    "01 · Multichain wallet",
    "01 · 多链钱包"
  ],
  "Ethereum、Arbitrum、Base、OP 的 Sepolia 測試網；另有 Polygon Amoy（POL）、Solana Devnet（SOL）與 TRON Shasta（TRX／TRC-20）。": [
    "Ethereum, Arbitrum, Base and OP Sepolia; Polygon Amoy (POL), Solana Devnet (SOL) and TRON Shasta (TRX / TRC-20).",
    "Ethereum、Arbitrum、Base、OP 的 Sepolia 测试网；另有 Polygon Amoy（POL）、Solana Devnet（SOL）与 TRON Shasta（TRX／TRC-20）。"
  ],
  "不建立錢包，先唯讀觀察 →": [
    "Watch an address without creating a wallet →",
    "不创建钱包，先唯读观察 →"
  ],
  "ETH↔USDC 引導、WETH 包裝／解包、交易池比較、有限額授權與撤銷。每筆交易都先核對。": [
    "Guided ETH ↔ USDC swaps, WETH wrapping, pool comparison, limited allowances and revocation. Review each transaction before signing.",
    "ETH↔USDC 引导、WETH 包装／解包、交易池比较、有限额授权与撤销。每笔交易都先核对。"
  ],
  "查看兌換流程 →": [
    "Explore the exchange →",
    "查看兑换流程 →"
  ],
  "03 · 故障恢復": [
    "03 · Recovery",
    "03 · 故障恢复"
  ],
  "EVM 簽名先保存再廣播；相同報價重送保持同一交易。可查核替換關係與鏈重組。": [
    "EVM signatures are persisted before broadcast. Retrying the same quote retains the transaction. Replacement relationships and reorgs can be inspected.",
    "EVM 签名先保存再广播；相同报价重送保持同一交易。可查核替换关系与链重组。"
  ],
  "查看交易診斷 →": [
    "Explore diagnostics →",
    "查看交易诊断 →"
  ],
  "04 · 可重現驗證": [
    "04 · Reproducible verification",
    "04 · 可重现验证"
  ],
  "Go Race Detector、Mock 故障注入、隔離瀏覽器測試。測試通過與真實上鏈證據分開呈現。": [
    "Go Race Detector, mock fault injection and isolated browser tests. Automated tests and real on-chain evidence are reported separately.",
    "Go Race Detector、Mock 故障注入、隔离浏览器测试。测试通过与真实上链证据分开呈现。"
  ],
  "設計與驗證文件 ↗": [
    "Design & verification documents ↗",
    "设计与验证文档 ↗"
  ],
  "交易怎麼處理": [
    "How transactions are handled",
    "交易如何处理"
  ],
  "畫面：核對交易內容與手續費": [
    "UI: review transaction details and fees",
    "页面：核对交易内容与手续费"
  ],
  "Go：驗證、模擬、綁定報價": [
    "Go: validate, simulate and bind quotes",
    "Go：验证、仿真、绑定报价"
  ],
  "本機加密金鑰：密碼解鎖簽署": [
    "Encrypted local keys: unlock with a password to sign",
    "本机加密密钥：密码解锁签署"
  ],
  "日誌：保存簽名與識別碼": [
    "Journal: persist signatures and identifiers",
    "日志：保存签名与识别码"
  ],
  "測試網 RPC：廣播與收據": [
    "Testnet RPC: broadcast and retrieve receipts",
    "测试网 RPC：广播与收据"
  ],
  "EVM 依帳戶、網路分開日誌；Solana 與 TRON 各有獨立日誌和交易有效期。RPC 報價及 finality 都是外部節點的觀測結果。": [
    "EVM journals are separated by account and network. Solana and TRON have their own journals and transaction validity windows. Quotes and finality are observations reported by external RPC nodes.",
    "EVM 依账户、网络分开日志；Solana 与 TRON 各有独立日志和交易有效期。RPC 报价及 finality 都是外部节点的观测结果。"
  ],
  "已完成的真實上鏈驗收": [
    "Verified on-chain transactions",
    "已完成的真实上链验收"
  ],
  "以下為已取得成功收據的實際交易，附區塊瀏覽器連結。這是驗收當下的快照，可用專案內的唯讀驗證工具重新查核。": [
    "These real transactions have successful receipts and explorer links. This is an acceptance snapshot; use the repository's read-only verification tool to check again.",
    "以下为已取得成功收据的实际交易，附区块浏览器链接。这是验收当下的快照，可用项目内的唯读验证工具重新查核。"
  ],
  "載入驗收紀錄…": [
    "Loading acceptance records…",
    "加载验收记录…"
  ],
  "本機交易證據": [
    "Local transaction evidence",
    "本机交易证据"
  ],
  "下方即時讀取所選 Ethereum Sepolia 帳戶的交易紀錄；只有實際收到的狀態會顯示。這份清單不是安全稽核或完整歷史索引。": [
    "Live transaction history for the selected Ethereum Sepolia account. Only observed states are shown. This is not a security audit or a complete history index.",
    "下方即时读取所选 Ethereum Sepolia 账户的交易记录；只有实际收到的状态会显示。这份清单不是安全稽核或完整历史索引。"
  ],
  "本機帳戶": [
    "Local account",
    "本机账户"
  ],
  "原有帳戶": [
    "Original account",
    "原有账户"
  ],
  "更新本機證據": [
    "Refresh local evidence",
    "更新本机证据"
  ],
  "已知測試入金：": [
    "Known test funding:",
    "已知测试入金："
  ],
  "0.05 Sepolia ETH 水龍頭交易 ↗": [
    "0.05 Sepolia ETH faucet transaction ↗",
    "0.05 Sepolia ETH 水龙头交易 ↗"
  ],
  "入金不等於已驗收本錢包的發送、授權或兌換。": [
    "Receiving funds does not validate this wallet's sending, approvals or swaps.",
    "入金不等于已验收本钱包的发送、授权或兑换。"
  ],
  "三分鐘展示路線": [
    "Three-minute demo",
    "三分钟展示路线"
  ],
  "0:00–0:40：產品定位、七個測試網與唯讀觀察。": [
    "0:00–0:40: Product scope, seven testnets and watch-only queries.",
    "0:00–0:40：产品定位、七个测试网与唯读观察。"
  ],
  "0:40–1:30：報價與交易確認，說明金額、手續費和必要授權。": [
    "0:40–1:30: Quotes, transaction review, amounts, fees and approvals.",
    "0:40–1:30：报价与交易确认，说明金额、手续费和必要授权。"
  ],
  "1:30–2:20：Exchange 引導、交易池比較及最低收到數量。": [
    "1:30–2:20: Guided exchange, pool comparison and minimum received.",
    "1:30–2:20：Exchange 引导、交易池比较及最低收到数量。"
  ],
  "2:20–3:00：收據證據、失敗恢復及設計取捨。": [
    "2:20–3:00: Receipts, recovery and design tradeoffs.",
    "2:20–3:00：收据证据、失败恢复及设计取舍。"
  ],
  "送出交易需要錢包密碼；錄影時請隱藏助記詞與密碼。": [
    "Sending a transaction requires the wallet password. Hide recovery phrases and passwords when recording.",
    "发送交易需要钱包密码；录像时请隐藏助记词与密码。"
  ],
  "可深入討論的設計": [
    "Design discussions",
    "可深入讨论的设计"
  ],
  "為什麼逾時不能當作交易沒有送出？": [
    "Why does a timeout not prove a transaction was never sent?",
    "为什么逾时不能当作交易没有送出？"
  ],
  "同一 Nonce 的多次替換如何判定勝出交易？": [
    "How do multiple replacements at the same nonce resolve?",
    "同一 Nonce 的多次替换如何判定胜出交易？"
  ],
  "如何避免慢查詢覆蓋較新的日誌狀態？": [
    "How do we prevent slow queries from overwriting newer journal state?",
    "如何避免慢查找覆盖较新的日志状态？"
  ],
  "L2 的額外費用與 Solana 的 blockhash 有效期如何處理？": [
    "How are L2 fees and Solana blockhash expiration handled?",
    "L2 的额外费用与 Solana 的 blockhash 有效期如何处理？"
  ],
  "在 RPC 不可用時，介面如何表達未知與過期資訊？": [
    "How should the UI represent unknown or stale data when RPC is unavailable?",
    "在 RPC 不可用时，接口如何表达未知与过期信息？"
  ],
  "僅支援測試網，可在本機操作或使用公開 Demo。代幣兌換限 Ethereum Sepolia；Solana 目前只支援 SOL，不支援 SPL 代幣、DEX 或跨鏈橋。各項功能的驗證結果請見對應紀錄。": [
    "Testnets only, available locally or through the public demo. Token swaps use Ethereum Sepolia; Solana currently supports SOL, without SPL tokens, DEX or bridging. See the corresponding records for verification results.",
    "仅支持测试网，可在本机操作或使用公开 Demo。代币兑换限 Ethereum Sepolia；Solana 目前只支持 SOL，不支持 SPL 代币、DEX 或跨链桥。各项功能的验证结果请见对应记录。"
  ],
  "Solana Devnet · 支援 SOL 收付款。": [
    "Solana Devnet · Send and receive SOL.",
    "Solana Devnet · 支持 SOL 收付款。"
  ],
  "僅使用測試資產": [
    "Test assets only",
    "仅使用测试资产"
  ],
  "建立或還原 Solana 錢包": [
    "Create or restore a Solana wallet",
    "创建或还原 Solana 钱包"
  ],
  "請使用專用測試助記詞。Solana 與 EVM 帳戶各自備份。": [
    "Use a dedicated test recovery phrase. Back up Solana and EVM accounts separately.",
    "请使用专用测试助记词。Solana 与 EVM 账户各自备份。"
  ],
  "還原助記詞（新建請留白）": [
    "Recovery phrase (leave blank to create)",
    "还原助记词（新建请留白）"
  ],
  "建立／還原測試錢包": [
    "Create / restore test wallet",
    "创建／还原测试钱包"
  ],
  "從 Multi-Chain Wallet Solana 加密備份還原": [
    "Restore a Multi-Chain Wallet Solana encrypted backup",
    "从 Multi-Chain Wallet Solana 加密备份还原"
  ],
  "備份檔案": [
    "Backup file",
    "备份文件"
  ],
  "原密碼": [
    "Old password",
    "原密码"
  ],
  "新密碼": [
    "New password",
    "新密码"
  ],
  "還原加密備份": [
    "Restore encrypted backup",
    "还原加密备份"
  ],
  "還原規格": [
    "Recovery specification",
    "还原规格"
  ],
  "BIP-39 英文助記詞，SLIP-0010 Ed25519，m/44'/501'/0'/0'；額外密語為空。不同派生路徑會得到不同地址。": [
    "BIP-39 English phrase, SLIP-0010 Ed25519, m/44'/501'/0'/0'; empty passphrase. Other derivation paths produce different addresses.",
    "BIP-39 英文助记词，SLIP-0010 Ed25519，m/44'/501'/0'/0'；额外密语为空。不同派生路径会得到不同地址。"
  ],
  "先離線備份這些單字": [
    "Back up these words offline",
    "先脱机备份这些单字"
  ],
  "僅顯示這一次。不要貼到 AI 對話或分享畫面。": [
    "Shown only once. Never paste them into an AI chat or share your screen.",
    "仅显示这一次。不要贴到 AI 对话或分享画面。"
  ],
  "我已依序離線備份": [
    "I have backed up the words in order",
    "我已依序脱机备份"
  ],
  "完成備份": [
    "Finish backup",
    "完成备份"
  ],
  "更新餘額與收據": [
    "Refresh balance & receipts",
    "更新余额与收据"
  ],
  "Solana 收款地址": [
    "Solana receiving address",
    "Solana 收款地址"
  ],
  "領取 Devnet SOL ↗": [
    "Get Devnet SOL ↗",
    "领取 Devnet SOL ↗"
  ],
  "發送 SOL": [
    "Send SOL",
    "发送 SOL"
  ],
  "SOL 數量": [
    "SOL amount",
    "SOL 数量"
  ],
  "交易處理方式": [
    "How transactions work",
    "交易处理方式"
  ],
  "每筆使用近期區塊雜湊，過期需重新報價。一次處理一筆交易，等待 finalized 後再送下一筆。": [
    "Each transaction uses a recent blockhash; request a new quote after expiration. One transaction at a time: wait for finalized before sending the next.",
    "每笔使用近期区块哈希，过期需重新报价。一次处理一笔交易，等待 finalized 后再送下一笔。"
  ],
  "填入 0.000001 SOL 自轉": [
    "Fill 0.000001 SOL self-transfer",
    "填入 0.000001 SOL 自转"
  ],
  "備份": [
    "Backup",
    "备份"
  ],
  "加密檔採 Multi-Chain Wallet Solana 格式。助記詞可用於本頁還原；此檔案不是 Ethereum Keystore V3。": [
    "Backups use the Multi-Chain Wallet Solana format. Restore with the recovery phrase on this page. This file is not an Ethereum Keystore V3 file.",
    "加密档采 Multi-Chain Wallet Solana 格式。助记词可用于本页还原；此文件不是 Ethereum Keystore V3。"
  ],
  "密碼": [
    "Password",
    "密码"
  ],
  "變更密碼": [
    "Change password",
    "变更密码"
  ],
  "領取 0.01 測試 SOL": [
    "Get 0.01 test SOL",
    "领取 0.01 测试 SOL"
  ],
  "直接向 Devnet 申請空投；供應或額度不足時可使用外部水龍頭。": [
    "Request a Devnet airdrop. Use an external faucet if supply or rate limits prevent it.",
    "直接向 Devnet 申请空投；供应或额度不足时可使用外部水龙头。"
  ],
  "確認 Devnet SOL 轉帳": [
    "Review Devnet SOL transfer",
    "确认 Devnet SOL 转账"
  ],
  "簽署並送出 Devnet": [
    "Sign & send to Devnet",
    "签署并送出 Devnet"
  ],
  "Solana 公開收款地址": [
    "Public Solana receiving address",
    "Solana 公开收款地址"
  ],
  "TRON Shasta · 支援 TRX 與 TRC-20 收付款。": [
    "TRON Shasta · Send and receive TRX and TRC-20.",
    "TRON Shasta · 支持 TRX 与 TRC-20 收付款。"
  ],
  "建立或還原 TRON 錢包": [
    "Create or restore a TRON wallet",
    "创建或还原 TRON 钱包"
  ],
  "請使用專用測試助記詞。TRON 與 EVM 帳戶各自備份。": [
    "Use a dedicated test recovery phrase. Back up TRON and EVM accounts separately.",
    "请使用专用测试助记词。TRON 与 EVM 账户各自备份。"
  ],
  "從 Multi-Chain Wallet TRON 加密備份還原": [
    "Restore a Multi-Chain Wallet TRON encrypted backup",
    "从 Multi-Chain Wallet TRON 加密备份还原"
  ],
  "BIP-39 英文助記詞，BIP-44 secp256k1，m/44'/195'/0'/0/0；額外密語為空。不同派生路徑會得到不同地址。": [
    "BIP-39 English phrase, BIP-44 secp256k1, m/44'/195'/0'/0/0; empty passphrase. Other derivation paths produce different addresses.",
    "BIP-39 英文助记词，BIP-44 secp256k1，m/44'/195'/0'/0/0；额外密语为空。不同派生路径会得到不同地址。"
  ],
  "TRON 收款地址": [
    "TRON receiving address",
    "TRON 收款地址"
  ],
  "領取 Shasta TRX ↗": [
    "Get Shasta TRX ↗",
    "领取 Shasta TRX ↗"
  ],
  "TRC-20 合約（發送 TRX 請留白）": [
    "TRC-20 contract (blank for TRX)",
    "TRC-20 合约（发送 TRX 请留白）"
  ],
  "查詢代幣與餘額": [
    "Check token & balance",
    "查找代币与余额"
  ],
  "數量": [
    "Amount",
    "数量"
  ],
  "每筆使用近期區塊雜湊，過期需重新報價。一次處理一筆交易，等待固化收據後再送下一筆。": [
    "Each transaction uses a recent block reference; request a new quote after expiration. One transaction at a time: wait for a solidified receipt before sending the next.",
    "每笔使用近期区块哈希，过期需重新报价。一次处理一笔交易，等待固化收据后再送下一笔。"
  ],
  "填入小額 TRX，另填收款地址": [
    "Fill a small TRX amount, then enter a recipient",
    "填入小额 TRX，另填收款地址"
  ],
  "加密檔採 Multi-Chain Wallet TRON 格式。助記詞可用於本頁還原；此檔案使用 Keystore V3 保存 TRON 私鑰，還原後以 TRON 地址顯示。": [
    "Backups use the Multi-Chain Wallet TRON format. The file stores a TRON private key in Keystore V3 format and displays a TRON address after restoration.",
    "加密档采 Multi-Chain Wallet TRON 格式。助记词可用于本页还原；此文件使用 Keystore V3 保存 TRON 私钥，还原后以 TRON 地址显示。"
  ],
  "領取 5 測試 TRX": [
    "Get 5 test TRX",
    "领取 5 测试 TRX"
  ],
  "由本機專用帳戶直接發放，每小時可領取一次。": [
    "Sent from a dedicated local account. One claim per hour.",
    "由本机专用账户直接发放，每小时可领取一次。"
  ],
  "確認 Shasta 轉帳": [
    "Review Shasta transfer",
    "确认 Shasta 转账"
  ],
  "Bandwidth 按全額消耗 TRX 預留，Energy 加入緩衝；實際扣費依鏈上收據。fee_limit 僅限制 Energy，不限制所有費用。": [
    "Bandwidth reserves assume full TRX consumption; Energy includes a buffer. Actual fees come from receipts. fee_limit caps Energy, not all fees.",
    "Bandwidth 按全额消耗 TRX 预留，Energy 加入缓冲；实际扣费依链上收据。fee_limit 仅限制 Energy，不限制所有费用。"
  ],
  "簽署並送出 Shasta": [
    "Sign & send to Shasta",
    "签署并送出 Shasta"
  ],
  "TRON 公開收款地址": [
    "Public TRON receiving address",
    "TRON 公开收款地址"
  ],
  "查詢失敗，請稍後重試。": [
    "Query failed. Please try again later.",
    "查找失败，请稍后重试。"
  ],
  "在 Etherscan 核對 ↗": [
    "Verify in explorer ↗",
    "在 Etherscan 核对 ↗"
  ],
  "複製交易雜湊": [
    "Copy transaction hash",
    "拷贝交易哈希"
  ],
  "已複製到剪貼簿": [
    "Copied to clipboard",
    "已拷贝到剪贴板"
  ],
  "無法存取剪貼簿，請選取完整文字後手動複製。": [
    "Clipboard access failed. Select and copy the full text manually.",
    "无法访问剪贴板，请选取完整文本后手动拷贝。"
  ],
  "查詢逾時，結果未知。請稍後重試。": [
    "Query timed out; the result is unknown. Try again later.",
    "查找逾时，结果未知。请稍后重试。"
  ],
  "無法連線，請確認網路後重試。": [
    "Unable to connect. Check your network and retry.",
    "无法连接，请确认网络后重试。"
  ],
  "請輸入完整地址：0x 加上 40 個十六進位字元（0–9、a–f）。": [
    "Enter a full address: 0x followed by 40 hexadecimal characters (0–9, a–f).",
    "请输入完整地址：0x 加上 40 个十六进位字符（0–9、a–f）。"
  ],
  "請輸入完整交易雜湊：0x 加上 64 個十六進位字元（0–9、a–f）。": [
    "Enter a full transaction hash: 0x followed by 64 hexadecimal characters (0–9, a–f).",
    "请输入完整交易哈希：0x 加上 64 个十六进位字符（0–9、a–f）。"
  ],
  "重新查詢": [
    "Retry query",
    "重新查找"
  ],
  "更新中…": [
    "Refreshing…",
    "更新中…"
  ],
  "區塊時間 ${time(data.blockTime)}": [
    "Block time ${time(data.blockTime)}",
    "区块时间 ${time(data.blockTime)}"
  ],
  "已連線": [
    "Connected",
    "已连接"
  ],
  "最後查核 ${time(data.checkedAt)}": [
    "Last checked ${time(data.checkedAt)}",
    "最后查核 ${time(data.checkedAt)}"
  ],
  "未驗證": [
    "Unverified",
    "未验证"
  ],
  "未取得最新區塊": [
    "Latest block unavailable",
    "未取得最新区块"
  ],
  "無法連線": [
    "Disconnected",
    "无法连接"
  ],
  "目前狀態未知": [
    "Current status unknown",
    "目前状态未知"
  ],
  "正在讀取 ${networkName} 餘額…": [
    "Loading ${networkName} balance…",
    "正在读取 ${networkName} 余额…"
  ],
  "完整數值": [
    "Full value",
    "完整数值"
  ],
  "查詢區塊": [
    "Queried block",
    "查找区块"
  ],
  "查核時間": [
    "Checked at",
    "查核时间"
  ],
  "餘額以本次查詢的區塊為準。重新查詢可取得最新數值。": [
    "The balance reflects the queried block. Refresh for the latest value.",
    "余额以本次查找的区块为准。重新查找可取得最新数值。"
  ],
  "正在取得交易收據…": [
    "Fetching transaction receipt…",
    "正在取得交易收据…"
  ],
  "鏈上執行成功": [
    "Executed successfully",
    "链上运行成功"
  ],
  "鏈上執行失敗": [
    "Execution failed",
    "链上运行失败"
  ],
  "等待區塊收錄": [
    "Awaiting block inclusion",
    "等待区块收录"
  ],
  "收據暫時不可用": [
    "Receipt temporarily unavailable",
    "收据暂时不可用"
  ],
  "區塊已變更，結果待確認": [
    "Block changed; result unconfirmed",
    "区块已变更，结果待确认"
  ],
  "結果未知": [
    "Unknown result",
    "结果未知"
  ],
  "區塊高度": [
    "Block height",
    "区块高度"
  ],
  "確認數": [
    "Confirmations",
    "确认数"
  ],
  "消耗 gas": [
    "Gas used",
    "消耗 gas"
  ],
  "實際手續費": [
    "Actual fee",
    "实际手续费"
  ],
  "查核時間 ${time(data.checkedAt)}。尚無可確認的執行結果，請稍後重新查詢。": [
    "Checked ${time(data.checkedAt)}. Execution cannot yet be confirmed. Please query again later.",
    "查核时间 ${time(data.checkedAt)}。尚无可确认的运行结果，请稍后重新查找。"
  ],
  "確認數不等於最終確定（finality）。這是本次查核的快照；重新查詢可更新結果。": [
    "Confirmations do not mean finality. This is a query snapshot; refresh for an updated result.",
    "确认数不等于最终确定（finality）。这是本次查核的快照；重新查找可更新结果。"
  ],
  "Multi-Chain Wallet · ${networkName} 錢包": [
    "Multi-Chain Wallet · ${networkName} Wallet",
    "Multi-Chain Wallet · ${networkName} 钱包"
  ],
  "正在申請測試幣…": [
    "Requesting test tokens…",
    "正在申请测试币…"
  ],
  "無法讀取領幣設定": [
    "Unable to load faucet settings",
    "无法读取领币设置"
  ],
  "領取失敗，請稍後再試": [
    "Claim failed. Please try again later",
    "领取失败，请稍后再试"
  ],
  "查看鏈上交易 ↗": [
    "View on-chain transaction ↗",
    "查看链上交易 ↗"
  ],
  "本小時已有同額發放紀錄，正在核對原交易。": [
    "A matching claim exists this hour. Checking the original transaction.",
    "本小时已有同额发放记录，正在核对原交易。"
  ],
  "申請已送出，等待鏈上結果。": [
    "Request submitted. Waiting for the on-chain result.",
    "申请已送出，等待链上结果。"
  ],
  "空投已提交，請核對最新餘額與交易連結；尚未確認到帳。": [
    "Airdrop submitted. Check your latest balance and transaction link; arrival is not yet confirmed.",
    "空投已提交，请核对最新余额与交易链接；尚未确认到帐。"
  ],
  "本小時已領取，原交易已成功；未重複發放，已重新查詢餘額。": [
    "Already claimed this hour. The original transfer succeeded; no duplicate was sent. Balance refreshed.",
    "本小时已领取，原交易已成功；未重复发放，已重新查找余额。"
  ],
  "測試幣轉帳已在鏈上執行成功，已重新查詢餘額。": [
    "Test token transfer executed successfully. Balance refreshed.",
    "测试币转账已在链上运行成功，已重新查找余额。"
  ],
  "這筆發放交易執行失敗，未完成領取。": [
    "This funding transaction failed; tokens were not delivered.",
    "这笔发放交易运行失败，未完成领取。"
  ],
  "交易結果仍待確認；請保留此連結，稍後更新餘額。": [
    "Result still unconfirmed. Keep this link and refresh your balance later.",
    "交易结果仍待确认；请保留此链接，稍后更新余额。"
  ],
  "連線中斷或逾時，空投結果未知。請先更新餘額，不要連續申請。": [
    "Connection lost or timed out; airdrop result unknown. Refresh your balance before requesting again.",
    "连接中断或逾时，空投结果未知。请先更新余额，不要连续申请。"
  ],
  "連線中斷或逾時，領取結果未知。請先更新餘額；再次申請會優先查找原發放交易。": [
    "Connection lost or timed out; claim result unknown. Refresh your balance. A retry will check for the original funding transaction first.",
    "连接中断或逾时，领取结果未知。请先更新余额；再次申请会优先查找原发放交易。"
  ],
  "正在查核…": [
    "Verifying…",
    "正在查核…"
  ],
  "目前尚無可列出的本機發送紀錄；不以 Mock 測試代替鏈上交易證據。": [
    "No local outgoing transactions yet. Mock tests are not on-chain evidence.",
    "目前尚无可列出的本机发送记录；不以 Mock 测试代替链上交易证据。"
  ],
  "無法更新證據：": [
    "Unable to refresh evidence:",
    "无法更新证据："
  ],
  "驗收紀錄無法載入": [
    "Unable to load acceptance records",
    "验收记录无法加载"
  ],
  "紀錄時間：${new Date(data.recordedAt).toLocaleString('zh-TW')} · ${new Set(data.transactions.map(tx => tx.network)).size} 個測試網": [
    "Recorded: ${new Date(data.recordedAt).toLocaleString('zh-TW')} · ${new Set(data.transactions.map(tx => tx.network)).size} testnets",
    "记录时间：${new Date(data.recordedAt).toLocaleString('zh-TW')} · ${new Set(data.transactions.map(tx => tx.network)).size} 个测试网"
  ],
  "轉帳": [
    "Transfer",
    "转账"
  ],
  "ETH 換成 WETH": [
    "Wrap ETH to WETH",
    "ETH 换成 WETH"
  ],
  "WETH 換回 ETH": [
    "Unwrap WETH to ETH",
    "WETH 换回 ETH"
  ],
  "授權 USDC": [
    "Approve USDC",
    "授权 USDC"
  ],
  "授權 WETH": [
    "Approve WETH",
    "授权 WETH"
  ],
  "USDC 轉帳": [
    "USDC transfer",
    "USDC 转账"
  ],
  "TRX 轉帳": [
    "TRX transfer",
    "TRX 转账"
  ],
  "測試 USDT 轉帳": [
    "Test USDT transfer",
    "测试 USDT 转账"
  ],
  "驗收時已取得終局確認": [
    "Finality observed at acceptance",
    "验收时已取得终局确认"
  ],
  "驗收時已取得成功收據；終局性請重新查核": [
    "Successful receipt observed at acceptance; recheck finality",
    "验收时已取得成功收据；终局性请重新查核"
  ],
  "尚未完成的驗收與功能": [
    "Pending acceptance & features",
    "尚未完成的验收与功能"
  ],
  "${account.id ? '獨立帳戶' : '原有帳戶'} · ${account.address || '尚未建立'}": [
    "${account.id ? '獨立帳戶' : '原有帳戶'} · ${account.address || '尚未建立'}",
    "${account.id ? '独立账户' : '原有账户'} · ${account.address || '尚未创建'}"
  ],
  "廣播結果待確認": [
    "Broadcast result unknown",
    "广播结果待确认"
  ],
  "已廣播": [
    "Broadcast",
    "已广播"
  ],
  "已處理，等待確認": [
    "Processed; awaiting confirmation",
    "已处理，等待确认"
  ],
  "已確認，等待 finalized": [
    "Confirmed; awaiting finalized",
    "已确认，等待 finalized"
  ],
  "已 finalized": [
    "Finalized",
    "已 finalized"
  ],
  "操作失敗": [
    "Operation failed",
    "操作失败"
  ],
  "查詢逾時，結果未知；請先更新原交易紀錄。": [
    "Query timed out; result unknown. Refresh the original transaction history first.",
    "查找逾时，结果未知；请先更新原交易记录。"
  ],
  "查詢 Slot": [
    "Queried slot",
    "查找 Slot"
  ],
  "無法取得最新餘額": [
    "Latest balance unavailable",
    "无法取得最新余额"
  ],
  "· 終局確認": [
    "· Finalized",
    "· 终局确认"
  ],
  "重新廣播原交易": [
    "Rebroadcast original transaction",
    "重新广播原交易"
  ],
  "未能更新收據；以下若有紀錄，僅為上次查核結果。": [
    "Receipts could not be refreshed. Any records below reflect the previous check.",
    "未能更新收据；以下若有记录，仅为上次查核结果。"
  ],
  "兩次密碼不同": [
    "Passwords do not match",
    "两次密码不同"
  ],
  "已複製": [
    "Copied",
    "已拷贝"
  ],
  "請手動複製完整地址": [
    "Please copy the full address manually",
    "请手动拷贝完整地址"
  ],
  "網路": [
    "Network",
    "网络"
  ],
  "收款人": [
    "Recipient",
    "收款人"
  ],
  "預估費用": [
    "Estimated fee",
    "预估费用"
  ],
  "有效至": [
    "Valid until",
    "有效至"
  ],
  "最晚有效區塊高度": [
    "Last valid block height",
    "最晚有效区块高度"
  ],
  "請選擇 8 KB 以內的備份檔": [
    "Choose a backup file no larger than 8 KB",
    "请选择 8 KB 以内的备份档"
  ],
  "兩次新密碼不同": [
    "New passwords do not match",
    "两次新密码不同"
  ],
  "密碼已更新，請重新下載備份；舊備份仍使用舊密碼。": [
    "Password updated. Download a new backup; older backups still use the old password.",
    "密码已更新，请重新下载备份；旧备份仍使用旧密码。"
  ],
  "已過期，固化鏈未查到收據": [
    "Expired; no receipt on the solidified chain",
    "已过期，固化链未查到收据"
  ],
  "可用 Bandwidth ${results[0].value.bandwidth} · Energy ${results[0].value.energy} · ${results[0].value.active ? '帳戶已啟用' : '尚未啟用，請先領取 TRX'}": [
    "Available Bandwidth ${results[0].value.bandwidth} · Energy ${results[0].value.energy} · ${results[0].value.active ? '帳戶已啟用' : '尚未啟用，請先領取 TRX'}",
    "可用 Bandwidth ${results[0].value.bandwidth} · Energy ${results[0].value.energy} · ${results[0].value.active ? '账户已激活' : '尚未激活，请先领取 TRX'}"
  ],
  "實際費用": [
    "Actual fee",
    "实际费用"
  ],
  "執行結果": [
    "Execution result",
    "运行结果"
  ],
  "Energy 用量估算": [
    "Estimated Energy",
    "Energy 用量估算"
  ],
  "Bandwidth 預留": [
    "Bandwidth reserve",
    "Bandwidth 预留"
  ],
  "代幣合約": [
    "Token contract",
    "代币合约"
  ],
  "原生 TRX": [
    "Native TRX",
    "原生 TRX"
  ],
  "${t.balance} ${t.symbol} · ${t.decimals} 位小數 · ${t.contract}": [
    "${t.balance} ${t.symbol} · ${t.decimals} decimals · ${t.contract}",
    "${t.balance} ${t.symbol} · ${t.decimals} 位小数 · ${t.contract}"
  ],
  "簽署並送出至 ${networkName}": [
    "Sign & send to ${networkName}",
    "签署并送出至 ${networkName}"
  ],
  "加速原交易": [
    "Speed up original transaction",
    "加速原交易"
  ],
  "取消原交易（零額自轉）": [
    "Cancel original (zero-value self-transfer)",
    "取消原交易（零额自转）"
  ],
  "資產轉帳": [
    "Asset transfer",
    "资产转账"
  ],
  "代幣轉帳": [
    "Token transfer",
    "代币转账"
  ],
  "代幣授權": [
    "Token approval",
    "代币授权"
  ],
  "ETH → WETH 包裝": [
    "ETH → WETH wrap",
    "ETH → WETH 包装"
  ],
  "WETH → ETH 解包": [
    "WETH → ETH unwrap",
    "WETH → ETH 解包"
  ],
  "代幣兌換": [
    "Token swap",
    "代币兑换"
  ],
  "同 Nonce 的另一筆交易已收錄": [
    "Another transaction at this nonce was included",
    "同 Nonce 的另一笔交易已收录"
  ],
  "已廣播，等待收錄": [
    "Broadcast; awaiting inclusion",
    "已广播，等待收录"
  ],
  "區塊變更，待確認": [
    "Block changed; unconfirmed",
    "区块变更，待确认"
  ],
  "收據尚不可用": [
    "Receipt unavailable",
    "收据尚不可用"
  ],
  "操作失敗，請稍後重試。": [
    "Operation failed. Please try again later.",
    "操作失败，请稍后重试。"
  ],
  "操作逾時，結果未知。請先更新錢包與交易紀錄，再決定是否重試。": [
    "Operation timed out; result unknown. Refresh your wallet and transaction history before retrying.",
    "操作逾时，结果未知。请先更新钱包与交易记录，再决定是否重试。"
  ],
  "正在查詢 ${networkName} 餘額…": [
    "Loading ${networkName} balance…",
    "正在查找 ${networkName} 余额…"
  ],
  "正在查詢鏈上餘額…": [
    "Loading on-chain balance…",
    "正在查找链上余额…"
  ],
  "區塊 ${balance.block} · ${time(balance.checkedAt)}": [
    "Block ${balance.block} · ${time(balance.checkedAt)}",
    "区块 ${balance.block} · ${time(balance.checkedAt)}"
  ],
  "已查到 ${balance.eth} ${networkName} ${nativeSymbol}。可先預估費用，是否足夠仍以報價為準。": [
    "Found ${balance.eth} ${networkName} ${nativeSymbol}. Request a quote to check whether this covers the transfer and fees.",
    "已查到 ${balance.eth} ${networkName} ${nativeSymbol}。可先预估费用，是否足够仍以报价为准。"
  ],
  "目前餘額為 0 ${nativeSymbol}。若剛領取，請稍後再查；代幣無法直接支付 gas。": [
    "Current balance is 0 ${nativeSymbol}. If you just claimed, check again later. Tokens cannot directly pay gas.",
    "目前余额为 0 ${nativeSymbol}。若刚领取，请稍后再查；代币无法直接支付 gas。"
  ],
  "無法查到最新餘額，目前無法確認測試幣是否到帳，請稍後重試。": [
    "Latest balance unavailable. Test funding arrival cannot be confirmed; try again later.",
    "无法查到最新余额，目前无法确认测试币是否到帐，请稍后重试。"
  ],
  "無法更新紀錄，請稍後再試。": [
    "Unable to refresh history. Please try again later.",
    "无法更新记录，请稍后再试。"
  ],
  "尚無交易。第一筆轉帳會出現在這裡。": [
    "No transactions yet. Your first transfer will appear here.",
    "尚无交易。第一笔转账会出现在这里。"
  ],
  "交易詳情": [
    "Transaction details",
    "交易详情"
  ],
  "資產操作": [
    "Asset action",
    "资产操作"
  ],
  "${tx.action === 'approve' ? '被授權地址' : '收款人'} ${tx.to}": [
    "${tx.action === 'approve' ? '被授權地址' : '收款人'} ${tx.to}",
    "${tx.action === 'approve' ? '被授权地址' : '收款人'} ${tx.to}"
  ],
  "狀態待確認": [
    "Unconfirmed status",
    "状态待确认"
  ],
  "已收錄的替代交易：": [
    "Included replacement transaction:",
    "已收录的替代交易："
  ],
  "已達鏈上終局性": [
    "Finality reached",
    "已达链上终局性"
  ],
  "${tx.confirmations} 次確認": [
    "${tx.confirmations} confirmations",
    "${tx.confirmations} 次确认"
  ],
  "實際費用 ${tx.feeEth} ${nativeSymbol}": [
    "Actual fee ${tx.feeEth} ${nativeSymbol}",
    "实际费用 ${tx.feeEth} ${nativeSymbol}"
  ],
  "收據顯示執行成功；代幣實際移動請核對合約紀錄。": [
    "Receipt reports success. Verify token movements against contract events.",
    "收据显示运行成功；代币实际移动请核对合约记录。"
  ],
  "加速交易": [
    "Speed up",
    "加速交易"
  ],
  "取消交易": [
    "Cancel transaction",
    "取消交易"
  ],
  "交易已記錄，結果待確認": [
    "Transaction recorded; result unconfirmed",
    "交易已记录，结果待确认"
  ],
  "已記錄交易雜湊。更新交易紀錄以確認收錄結果；廣播成功不等於交易執行成功。": [
    "Transaction hash saved. Refresh history to verify inclusion. Broadcast success does not mean execution success.",
    "已记录交易哈希。更新交易记录以确认收录结果；广播成功不等于交易运行成功。"
  ],
  "領取 ${walletState.chainId === 11155111 ? '0.001' : walletState.chainId === 80002 ? '0.1' : '0.0001'} 測試 ${nativeSymbol}": [
    "Get ${walletState.chainId === 11155111 ? '0.001' : walletState.chainId === 80002 ? '0.1' : '0.0001'} test ${nativeSymbol}",
    "领取 ${walletState.chainId === 11155111 ? '0.001' : walletState.chainId === 80002 ? '0.1' : '0.0001'} 测试 ${nativeSymbol}"
  ],
  "領取測試 POL ↗": [
    "Get test POL ↗",
    "领取测试 POL ↗"
  ],
  "僅接收 ${networkName} 的 ${nativeSymbol} 與代幣。": [
    "Only receive ${networkName} ${nativeSymbol} and tokens.",
    "仅接收 ${networkName} 的 ${nativeSymbol} 与代币。"
  ],
  "還原測試錢包": [
    "Restore test wallet",
    "还原测试钱包"
  ],
  "密碼長度必須介於 12 至 128 字元": [
    "Password must be 12–128 characters",
    "密码长度必须介于 12 至 128 字符"
  ],
  "兩次密碼不同，請重新確認。": [
    "Passwords do not match. Please check again.",
    "两次密码不同，请重新确认。"
  ],
  "正在加密金鑰，請稍候…": [
    "Encrypting keys. Please wait…",
    "正在加密密钥，请稍候…"
  ],
  "無法複製，請選取上方完整地址手動複製。": [
    "Unable to copy. Select the full address above and copy it manually.",
    "无法拷贝，请选取上方完整地址手动拷贝。"
  ],
  "地址已複製。請在 Google 水龍頭貼上地址並申請；完成後按「我已領取，查詢餘額」。": [
    "Address copied. Paste it into the Google faucet and claim, then check your balance.",
    "地址已拷贝。请在 Google 水龙头粘贴地址并申请；完成后按「我已领取，查找余额」。"
  ],
  "無法自動複製，請手動複製上方收款地址，在 Google 水龍頭貼上並申請。": [
    "Automatic copy failed. Copy the receiving address above and paste it into the Google faucet.",
    "无法自动拷贝，请手动拷贝上方收款地址，在 Google 水龙头粘贴并申请。"
  ],
  "正在驗證密碼…": [
    "Verifying password…",
    "正在验证密码…"
  ],
  "已要求瀏覽器下載加密備份，請妥善保存檔案與密碼。": [
    "Backup download requested. Keep the file and password safe.",
    "已要求浏览器下载加密备份，请妥善保存盘案与密码。"
  ],
  "餘額未更新，顯示上次查詢結果。": [
    "Balance is stale; showing the previous query.",
    "余额未更新，显示上次查找结果。"
  ],
  "代幣詳情": [
    "Token details",
    "代币详情"
  ],
  "精度": [
    "Decimals",
    "精度"
  ],
  "最小單位餘額": [
    "Balance in base units",
    "最小单位余额"
  ],
  "${item.symbol} · ${item.contract.slice(0, 8)}…${item.stale ? '（餘額未更新）' : ''}": [
    "${item.symbol} · ${item.contract.slice(0, 8)}…${item.stale ? '（餘額未更新）' : ''}",
    "${item.symbol} · ${item.contract.slice(0, 8)}…${item.stale ? '（余额未更新）' : ''}"
  ],
  "被授權地址（spender）": [
    "Spender address",
    "被授权地址（spender）"
  ],
  "只授權指定數量；輸入 0 代表撤銷。手續費以 ${nativeSymbol} 支付。": [
    "Approve only the specified amount; 0 revokes approval. Fees are paid in ${nativeSymbol}.",
    "只授权指定数量；输入 0 代表撤销。手续费以 ${nativeSymbol} 支付。"
  ],
  "手續費另以 ${nativeSymbol} 支付。": [
    "Fees are paid separately in ${nativeSymbol}.",
    "手续费另以 ${nativeSymbol} 支付。"
  ],
  "此代幣餘額未更新，預估費用時會重新查核。": [
    "This token balance is stale and will be checked again when quoting.",
    "此代币余额未更新，预估费用时会重新查核。"
  ],
  "正在預估與模擬…": [
    "Estimating and simulating…",
    "正在预估与仿真…"
  ],
  "發送地址": [
    "Sender",
    "发送地址"
  ],
  "被授權地址": [
    "Spender",
    "被授权地址"
  ],
  "執行費用上限": [
    "Execution fee cap",
    "运行费用上限"
  ],
  "預估總扣款（含費用預留）": [
    "Estimated total debit (including fee reserve)",
    "预估总扣款（含费用预留）"
  ],
  "最多扣除": [
    "Maximum debit",
    "最多扣除"
  ],
  "報價有效至": [
    "Quote valid until",
    "报价有效至"
  ],
  "L1／營運費預留": [
    "L1 / operator fee reserve",
    "L1／营运费预留"
  ],
  "${data.rollupFeeEth} ETH（估算含緩衝；上鏈費用仍可能變動）": [
    "${data.rollupFeeEth} ETH (buffered estimate; on-chain fees may change)",
    "${data.rollupFeeEth} ETH（估算含缓冲；上链费用仍可能变动）"
  ],
  "預估收到": [
    "Estimated received",
    "预估收到"
  ],
  "最低收到": [
    "Minimum received",
    "最低收到"
  ],
  "收到資產": [
    "Receiving asset",
    "收到资产"
  ],
  "兌換合約": [
    "Swap contract",
    "兑换合约"
  ],
  "滑價": [
    "Slippage",
    "滑价"
  ],
  "交易截止時間": [
    "Transaction deadline",
    "交易截止时间"
  ],
  "執行費有簽署上限；L1／營運費為預留估算，無法由這筆交易設定絕對上限。費用變動需重新預估。": [
    "Execution fees have a signed cap. L1 / operator fees are estimates without an absolute cap in this transaction. Requote if fees change.",
    "运行费有签署上限；L1／营运费为预留估算，无法由这笔交易设置绝对上限。费用变动需重新预估。"
  ],
  "使用相同 Nonce 與較高手續費競爭收錄。原交易仍可能先成功；送出取消不代表已取消。": [
    "Uses the same nonce and higher fees to compete for inclusion. The original may succeed first; submitting a cancel does not guarantee cancellation.",
    "使用相同 Nonce 与较高手续费竞争收录。原交易仍可能先成功；送出取消不代表已取消。"
  ],
  "檢視實際簽署內容": [
    "Inspect signed content",
    "查看实际签署内容"
  ],
  "最小單位": [
    "Base units",
    "最小单位"
  ],
  "方法": [
    "Method",
    "方法"
  ],
  "交易池": [
    "Pool",
    "交易池"
  ],
  "報價已過期，請取消並重新預估。": [
    "Quote expired. Cancel and request a new quote.",
    "报价已过期，请取消并重新预估。"
  ],
  "正在簽署與廣播…": [
    "Signing and broadcasting…",
    "正在签署与广播…"
  ],
  "部分代幣餘額無法更新，請稍後重試。": [
    "Some token balances could not be refreshed. Try again later.",
    "部分代币余额无法更新，请稍后重试。"
  ],
  "Sepolia Router：${config.router}。收到的資產回到本錢包。": [
    "Sepolia Router: ${config.router}. Received assets return to this wallet.",
    "Sepolia Router：${config.router}。收到的资产回到本钱包。"
  ],
  "Sepolia WETH：${config.weth}。包裝／解包比率 1:1，另付 ETH gas。": [
    "Sepolia WETH: ${config.weth}. Wrap / unwrap at 1:1, plus ETH gas.",
    "Sepolia WETH：${config.weth}。包装／解包比率 1:1，另付 ETH gas。"
  ],
  "正在查詢鏈上餘額與授權…": [
    "Checking on-chain balance and allowance…",
    "正在查找链上余额与授权…"
  ],
  "請填入支付數量，再查詢下一步。": [
    "Enter an amount before checking the next step.",
    "请填入支付数量，再查找下一步。"
  ],
  "支付餘額不足，請先收款或包裝 ETH。": [
    "Insufficient balance. Receive tokens or wrap ETH first.",
    "支付余额不足，请先收款或包装 ETH。"
  ],
  "額度足夠，可直接取得兌換報價。": [
    "Allowance is sufficient. You can request a swap quote.",
    "额度足够，可直接取得兑换报价。"
  ],
  "額度不足；先撤銷既有授權，等待成功後授權本次數量。": [
    "Insufficient allowance. Revoke the existing allowance, wait for success, then approve this amount.",
    "额度不足；先撤销既有授权，等待成功后授权本次数量。"
  ],
  "請授權本次數量，等待交易成功後取得兌換報價。": [
    "Approve this amount and wait for success before requesting a swap quote.",
    "请授权本次数量，等待交易成功后取得兑换报价。"
  ],
  "可用 ${token.balance} ${token.symbol} · 目前授權 ${token.allowance} ${token.symbol}。${next} 每次送出前仍會重新查核。": [
    "Available ${token.balance} ${token.symbol} · Allowance ${token.allowance} ${token.symbol}. ${next} Checks run again before every send.",
    "可用 ${token.balance} ${token.symbol} · 目前授权 ${token.allowance} ${token.symbol}。${next} 每次送出前仍会重新查核。"
  ],
  "查詢失敗：${error.message}": [
    "Query failed: ${error.message}",
    "查找失败：${error.message}"
  ],
  "數量已變更，請重新查詢餘額與授權。": [
    "Amount changed. Check balance and allowance again.",
    "数量已变更，请重新查找余额与授权。"
  ],
  "正在比較四個費率的鏈上報價…": [
    "Comparing on-chain quotes across four fee tiers…",
    "正在比较四个费率的链上报价…"
  ],
  "輸入已變更，請重新比較。": [
    "Input changed. Compare again.",
    "输入已变更，请重新比较。"
  ],
  "依預估收到數量排序選池；未扣除 gas，各池查詢時間可能不同。送出前會重新報價與模擬。": [
    "Pools are ranked by estimated output before gas. Quotes may come from different times. The transaction is requoted and simulated before sending.",
    "依预估收到数量排序选池；未扣除 gas，各池查找时间可能不同。送出前会重新报价与仿真。"
  ],
  "${pool.fee/10000}% · ${pool.error || pool.output+' '+result.symbol}${pool.fee===result.bestFee?' · 本次輸出最高':''}": [
    "${pool.fee/10000}% · ${pool.error || pool.output+' '+result.symbol}${pool.fee===result.bestFee?' · 本次輸出最高':''}",
    "${pool.fee/10000}% · ${pool.error || pool.output+' '+result.symbol}${pool.fee===result.bestFee?' · 本次输出最高':''}"
  ],
  "目前沒有可用的交易池報價。": [
    "No pool quote is currently available.",
    "目前没有可用的交易池报价。"
  ],
  "瀏覽器無法保存引導進度；請保留交易雜湊，重新整理後從歷史確認。": [
    "Unable to save guide progress in this browser. Keep your transaction hash and verify history after refreshing.",
    "浏览器无法保存引导进度；请保留交易哈希，刷新后从历史确认。"
  ],
  "包裝 ETH 成為 WETH": [
    "Wrap ETH into WETH",
    "包装 ETH 成为 WETH"
  ],
  "查核授權並兌換": [
    "Check allowance and swap",
    "查核授权并兑换"
  ],
  "將本次收到的 WETH 解包成 ETH": [
    "Unwrap the WETH received in this swap",
    "将本次收到的 WETH 解包成 ETH"
  ],
  "已完成本次引導；收據仍可在交易紀錄核對": [
    "Guide completed. Verify receipts in transaction history",
    "已完成本次引导；收据仍可在交易记录核对"
  ],
  "${flow.direction==='eth-usdc'?'ETH → USDC':'USDC → ETH'} · ${flow.pending?'等待 '+(flow.pending.hash||'原報價')+' 的鏈上收據':phases[flow.phase]}": [
    "${flow.direction==='eth-usdc'?'ETH → USDC':'USDC → ETH'} · ${flow.pending?'等待 '+(flow.pending.hash||'原報價')+' 的鏈上收據':phases[flow.phase]}",
    "${flow.direction==='eth-usdc'?'ETH → USDC':'USDC → ETH'} · ${flow.pending?'等待 '+(flow.pending.hash||'原报价')+' 的链上收据':phases[flow.phase]}"
  ],
  "選擇 ETH ↔ USDC 時，系統會依序準備必要交易；每筆皆需獨立確認與簽署。": [
    "The ETH ↔ USDC guide prepares the required transactions in order. Review and sign each transaction separately.",
    "选择 ETH ↔ USDC 时，系统会依序准备必要交易；每笔皆需独立确认与签署。"
  ],
  "更新進度": [
    "Refresh progress",
    "更新进度"
  ],
  "繼續下一步並核對": [
    "Review next step",
    "继续下一步并核对"
  ],
  "尚未找到原報價的交易紀錄。請更新進度，或在原確認視窗重試同一筆報價。": [
    "No transaction was found for the original quote. Refresh progress or retry the same quote in the original confirmation dialog.",
    "尚未找到原报价的交易记录。请更新进度，或在原确认窗口重试同一笔报价。"
  ],
  "此步驟執行失敗，已消耗測試 gas。可結束引導或重新預估。": [
    "This step failed and consumed test gas. End the guide or request a new quote.",
    "此步骤运行失败，已消耗测试 gas。可结束引导或重新预估。"
  ],
  "尚未取得可驗證的兌換收支。": [
    "Verifiable swap movements are not available yet.",
    "尚未取得可验证的兑换收支。"
  ],
  "未找到本次收到的 WETH，請核對收據後手動解包。": [
    "The WETH received in this swap was not found. Verify the receipt before unwrapping manually.",
    "未找到本次收到的 WETH，请核对收据后手动解包。"
  ],
  "尚無法確認此步驟結果，保留原交易，請稍後更新。": [
    "This step is still unconfirmed. The original transaction is retained; refresh later.",
    "尚无法确认此步骤结果，保留原交易，请稍后更新。"
  ],
  "請輸入大於 0 的數量。": [
    "Enter an amount greater than 0.",
    "请输入大于 0 的数量。"
  ],
  "支付數量超過代幣精度。": [
    "Amount exceeds token precision.",
    "支付数量超过代币精度。"
  ],
  "來源代幣餘額不足。": [
    "Insufficient source token balance.",
    "来源代币余额不足。"
  ],
  "引導已變更，請重新報價。": [
    "Guide changed. Request a new quote.",
    "引导已变更，请重新报价。"
  ],
  "測試 USDC": [
    "Test USDC",
    "测试 USDC"
  ],
  "${raw} 最小單位 · ${asset}": [
    "${raw} base units · ${asset}",
    "${raw} 最小单位 · ${asset}"
  ],
  "正在查核本頁鏈上收據…": [
    "Verifying receipts on this page…",
    "正在查核本页链上收据…"
  ],
  "第 ${data.page} / ${data.pages} 頁 · 已收錄 ${data.totalTransactions} 筆": [
    "Page ${data.page} / ${data.pages} · ${data.totalTransactions} indexed",
    "第 ${data.page} / ${data.pages} 页 · 已收录 ${data.totalTransactions} 笔"
  ],
  "尚無收支。取得測試 ${nativeSymbol} 後，可同步最近區塊或匯入收款交易。": [
    "No activity yet. After receiving test ${nativeSymbol}, sync recent blocks or import the receiving transaction.",
    "尚无收支。取得测试 ${nativeSymbol} 后，可同步最近区块或导入收款交易。"
  ],
  "部分交易尚未確認或無法查核，不列入本頁合計。": [
    "Some transactions are unconfirmed or cannot be verified and are excluded from this page's totals.",
    "部分交易尚未确认或无法查核，不列入本页合计。"
  ],
  "本頁 ${nativeSymbol} 收支": [
    "${nativeSymbol} activity on this page",
    "本页 ${nativeSymbol} 收支"
  ],
  "本頁代幣 ${total.asset}": [
    "Token on this page: ${total.asset}",
    "本页代币 ${total.asset}"
  ],
  "收入 ${displayAsset(total.receivedRaw,total.asset)} · 支出 ${displayAsset(total.sentRaw,total.asset)}": [
    "Received ${displayAsset(total.receivedRaw,total.asset)} · Sent ${displayAsset(total.sentRaw,total.asset)}",
    "收入 ${displayAsset(total.receivedRaw,total.asset)} · 支出 ${displayAsset(total.sentRaw,total.asset)}"
  ],
  "手續費 ${displayAsset(total.feeRaw,total.asset)} · 淨變動 ${displayAsset(total.netRaw,total.asset)}": [
    "Fees ${displayAsset(total.feeRaw,total.asset)} · Net ${displayAsset(total.netRaw,total.asset)}",
    "手续费 ${displayAsset(total.feeRaw,total.asset)} · 净变动 ${displayAsset(total.netRaw,total.asset)}"
  ],
  "付款": [
    "Send",
    "付款"
  ],
  "手續費": [
    "Fee",
    "手续费"
  ],
  "無法查核": [
    "Unable to verify",
    "无法查核"
  ],
  "${time(tx.blockTime)} · 區塊 ${tx.block}": [
    "${time(tx.blockTime)} · Block ${tx.block}",
    "${time(tx.blockTime)} · 区块 ${tx.block}"
  ],
  "尚無可列入的資產變動。授權操作本身不算代幣支出。": [
    "No eligible asset movements. Approval itself is not a token expense.",
    "尚无可列入的资产变动。授权操作本身不算代币支出。"
  ],
  "對方地址：${movement.counterparty}": [
    "Counterparty: ${movement.counterparty}",
    "对方地址：${movement.counterparty}"
  ],
  "鏈上紀錄詳情": [
    "On-chain record details",
    "链上记录详情"
  ],
  "紀錄來源": [
    "Record source",
    "记录来源"
  ],
  "原始數量": [
    "Raw amount",
    "原始数量"
  ],
  "已查核並收錄 ${result.hash}。重複匯入不會重複計帳。": [
    "Verified and indexed ${result.hash}. Re-importing will not duplicate activity.",
    "已查核并收录 ${result.hash}。重复导入不会重复计帐。"
  ],
  "正在掃描區塊，請稍候…": [
    "Scanning blocks. Please wait…",
    "正在扫描区块，请稍候…"
  ],
  "請輸入有效的區塊號碼。": [
    "Enter a valid block number.",
    "请输入有效的区块号码。"
  ],
  "已同步區塊 ${result.from}–${result.to}，新增 ${result.added} 筆。可保留下一個起始區塊繼續同步，或清空改查最近區塊。": [
    "Synced blocks ${result.from}–${result.to}; added ${result.added} records. Keep the next start block to continue or clear it to scan recent blocks.",
    "已同步区块 ${result.from}–${result.to}，添加 ${result.added} 笔。可保留下一个起始区块继续同步，或清空改查最近区块。"
  ],
  "請選擇不超過 8 KB 的加密備份。": [
    "Choose an encrypted backup no larger than 8 KB.",
    "请选择不超过 8 KB 的加密备份。"
  ],
  "新密碼需為 12–128 字元。": [
    "New password must be 12–128 characters.",
    "新密码需为 12–128 字符。"
  ],
  "兩次新密碼不一致。": [
    "New passwords do not match.",
    "两次新密码不一致。"
  ],
  "密碼已更新，請重新下載加密備份。": [
    "Password updated. Download a new encrypted backup.",
    "密码已更新，请重新下载加密备份。"
  ],
  "${state.enabled ? '同步已啟用' : '同步已暫停'} · 起點 ${state.start} · 下一區塊 ${state.next} · finalized ${state.finalized}${state.error ? ' · ' + state.error : ''}": [
    "${state.enabled ? '同步已啟用' : '同步已暫停'} · Start ${state.start} · Next block ${state.next} · finalized ${state.finalized}${state.error ? ' · ' + state.error : ''}",
    "${state.enabled ? '同步已激活' : '同步已暂停'} · 起点 ${state.start} · 下一区块 ${state.next} · finalized ${state.finalized}${state.error ? ' · ' + state.error : ''}"
  ],
  "起始區塊無效": [
    "Invalid start block",
    "起始区块无效"
  ],
  "請輸入完整且非零的地址；轉帳時仍會驗證大小寫校驗。": [
    "Enter a complete, nonzero address. Address checksum is still verified before transfer.",
    "请输入完整且非零的地址；转账时仍会验证大小写校验。"
  ],
  "移除": [
    "Remove",
    "移除"
  ],
  "瀏覽器無法儲存，地址簿尚未變更。": [
    "Browser storage is unavailable. The address book was not changed.",
    "浏览器无法保存，地址簿尚未变更。"
  ],
  "名稱需為 1–40 字元。": [
    "Name must be 1–40 characters.",
    "名称需为 1–40 字符。"
  ],
  "地址簿最多 100 筆。": [
    "The address book supports up to 100 entries.",
    "地址簿最多 100 笔。"
  ],
  "已儲存。": [
    "Saved.",
    "已保存。"
  ],
  "無法儲存變更。": [
    "Unable to save changes.",
    "无法保存变更。"
  ],
  "最多 20 個觀察地址。": [
    "Up to 20 watch addresses.",
    "最多 20 个观察地址。"
  ],
  "正在查詢…": [
    "Querying…",
    "正在查找…"
  ],
  "代幣查詢失敗：": [
    "Token query failed:",
    "代币查找失败："
  ],
  "正在核對收據…": [
    "Verifying receipt…",
    "正在核对收据…"
  ],
  "${tx.state} · ${tx.blockTime ? time(tx.blockTime) : '尚無區塊時間'}": [
    "${tx.state} · ${tx.blockTime ? time(tx.blockTime) : '尚無區塊時間'}",
    "${tx.state} · ${tx.blockTime ? time(tx.blockTime) : '尚无区块时间'}"
  ],
  "方向": [
    "Direction",
    "方向"
  ],
  "原始整數數量": [
    "Raw integer amount",
    "原始整数数量"
  ],
  "證據": [
    "Evidence",
    "证据"
  ],
  "尚無可列入的資產移動。": [
    "No eligible asset movements.",
    "尚无可列入的资产移动。"
  ],
  "正在查核指定區塊…": [
    "Inspecting the selected block…",
    "正在查核指定区块…"
  ],
  "區塊 ${data.block} · ${data.coverage}": [
    "Block ${data.block} · ${data.coverage}",
    "区块 ${data.block} · ${data.coverage}"
  ],
  "此區塊沒有找到相關交易；這不代表此地址沒有其他歷史。": [
    "No matching transactions in this block. This does not mean the address has no other history.",
    "此区块没有找到相关交易；这不代表此地址没有其他历史。"
  ],
  "鏈 ID": [
    "Chain ID",
    "链 ID"
  ],
  "查詢失敗": [
    "Query failed",
    "查找失败"
  ],
  "使用端點": [
    "Endpoint",
    "使用端点"
  ],
  "最近請求毫秒": [
    "Latest request (ms)",
    "最近请求毫秒"
  ],
  "請求數": [
    "Requests",
    "请求数"
  ],
  "傳輸失敗數": [
    "Transport failures",
    "传输失败数"
  ],
  "備援切換數": [
    "Failovers",
    "备援切换数"
  ],
  "已查到交易": [
    "Transaction found",
    "已查到交易"
  ],
  "已收錄區塊": [
    "Included block",
    "已收录区块"
  ],
  "等待可驗證收據": [
    "Awaiting verifiable receipt",
    "等待可验证收据"
  ],
  "執行成功": [
    "Execution succeeded",
    "运行成功"
  ],
  "執行失敗": [
    "Execution failed",
    "运行失败"
  ],
  "執行結果待確認": [
    "Execution unconfirmed",
    "运行结果待确认"
  ],
  "等待 finalized": [
    "Awaiting finalized",
    "等待 finalized"
  ],
  "區塊雜湊": [
    "Block hash",
    "区块哈希"
  ],
  "尚無": [
    "None yet",
    "尚无"
  ],
  "偵測到鏈重組，原收據不能作為目前執行結果。": [
    "Reorg detected. The original receipt cannot establish the current execution result.",
    "侦测到链重组，原收据不能作为目前运行结果。"
  ],
  "淺色模式": [
    "Light mode",
    "浅色模式"
  ],
  "獨立帳戶": [
    "Independent account",
    "独立账户"
  ],
  "帳戶": [
    "Account",
    "账户"
  ],
  "尚未建立": [
    "Not created",
    "尚未创建"
  ],
  "尚未建立錢包": [
    "Wallet not created",
    "尚未创建钱包"
  ],
  "帳戶已啟用": [
    "Account activated",
    "账户已激活"
  ],
  "尚未啟用，請先領取 TRX": [
    "Inactive; get test TRX first",
    "尚未激活，请先领取 TRX"
  ],
  "（餘額未更新）": [
    "(stale balance)",
    "（余额未更新）"
  ],
  "同步已啟用": [
    "Sync enabled",
    "同步已激活"
  ],
  "同步已暫停": [
    "Sync paused",
    "同步已暂停"
  ],
  "尚無區塊時間": [
    "Block time unavailable",
    "尚无区块时间"
  ],
  "等待": [
    "Waiting for",
    "等待"
  ],
  "原報價": [
    "Original quote",
    "原报价"
  ],
  "的鏈上收據": [
    "on-chain receipt",
    "的链上收据"
  ],
  "本次輸出最高": [
    "Highest quoted output",
    "本次输出最高"
  ],
  "錢包已存在，無法覆寫": [
    "Wallet already exists and cannot be overwritten",
    "钱包已存在，无法覆写"
  ],
  "尚未建立或匯入錢包": [
    "Create or import a wallet first",
    "尚未创建或导入钱包"
  ],
  "密碼錯誤，無法解密金鑰": [
    "Incorrect password; unable to decrypt the key",
    "密码错误，无法解密密钥"
  ],
  "助記詞格式不正確或校驗失敗": [
    "Invalid recovery phrase or checksum",
    "助记词格式不正确或校验失败"
  ],
  "地址格式不正確，請輸入 0x 開頭的 40 位十六進位地址": [
    "Invalid address. Enter 0x followed by 40 hexadecimal characters",
    "地址格式不正确，请输入 0x 开头的 40 位十六进位地址"
  ],
  "不可使用零地址": [
    "The zero address is not allowed",
    "不可使用零地址"
  ],
  "地址混合大小寫校驗和不正確": [
    "Mixed-case address checksum is invalid",
    "地址混合大小写校验和不正确"
  ],
  "RPC 連到其他網路，已停止操作；Multi-Chain Wallet 僅允許支援的測試網": [
    "RPC network mismatch. Operation stopped; only supported testnets are allowed",
    "RPC 连到其他网络，已停止操作；Multi-Chain Wallet 仅允许支持的测试网"
  ],
  "找不到指定的報價或報價已過期": [
    "Quote not found or expired",
    "找不到指定的报价或报价已过期"
  ],
  "報價已過期，請重新建立報價": [
    "Quote expired. Request a new quote",
    "报价已过期，请重新创建报价"
  ],
  "報價數量已達上限 (256)，請稍後重試": [
    "Quote limit reached (256). Try again later",
    "报价数量已达上限 (256)，请稍后重试"
  ],
  "已有處理中或廣播結果未確認的交易，請等待該交易確認後再操作": [
    "A transaction is pending or its broadcast is unconfirmed. Wait for confirmation first",
    "已有处理中或广播结果未确认的交易，请等待该交易确认后再操作"
  ],
  "鏈上 Nonce 已變更，請重新建立報價": [
    "On-chain nonce changed. Request a new quote",
    "链上 Nonce 已变更，请重新创建报价"
  ],
  "餘額不足以支付轉帳金額與最高 Gas 手續費": [
    "Insufficient balance for the transfer and maximum gas fees",
    "余额不足以支付转账金额与最高 Gas 手续费"
  ],
  "指定合約地址在目前測試網上沒有 bytecode": [
    "No contract bytecode exists at this address on the selected testnet",
    "指定合约地址在目前测试网上没有 bytecode"
  ],
  "既有授權額度大於 0，為避免 ERC20 approve race pattern，必須先將授權額度歸零（approve 0）後才能設定新的非零額度": [
    "An allowance already exists. Revoke it with approve 0 before setting another nonzero allowance",
    "既有授权额度大于 0，为避免 ERC20 approve race pattern，必须先将授权额度归零（approve 0）后才能设置新的非零额度"
  ],
  "不支援無上限授權，請輸入明確的授權額度": [
    "Unlimited approval is not supported. Enter an explicit allowance",
    "不支持无上限授权，请输入明确的授权额度"
  ],
  "交易模擬執行失敗（eth_call 未通過）": [
    "Transaction simulation failed (eth_call)",
    "交易仿真运行失败（eth_call 未通过）"
  ],
  "交易日誌數量已達上限 (1000)，為保全歷史紀錄已拒絕新交易": [
    "Journal limit reached (1000). New transactions are blocked to preserve history",
    "交易日志数量已达上限 (1000)，为保全历史记录已拒绝新交易"
  ],
  "代幣小數位數超出上限 36": [
    "Token decimals exceed the limit of 36",
    "代币小数字数超出上限 36"
  ],
  "代幣符號長度超出上限 32 字元": [
    "Token symbol exceeds 32 characters",
    "代币符号长度超出上限 32 字符"
  ],
  "系統密碼運算繁忙，請稍後重試": [
    "Password processing is busy. Try again later",
    "系统密码运算繁忙，请稍后重试"
  ],
  "另一筆測試幣申請正在處理，請稍後再試": [
    "Another test token claim is in progress. Try again later",
    "另一笔测试币申请正在处理，请稍后再试"
  ],
  "尚未設定 Shasta 測試幣發放帳戶，請使用外部水龍頭": [
    "No Shasta funding account is configured. Use an external faucet",
    "尚未设置 Shasta 测试币发放账户，请使用外部水龙头"
  ],
  "本小時已達測試幣發放上限": [
    "Hourly test token distribution limit reached",
    "本小时已达测试币发放上限"
  ],
  "Shasta 測試幣庫存不足，請先補充發放帳戶或使用外部水龍頭": [
    "Shasta faucet inventory is insufficient. Refill the funding account or use an external faucet",
    "Shasta 测试币库存不足，请先补充发放账户或使用外部水龙头"
  ],
  "Shasta 發放手續費超過 2 TRX，請稍後再試": [
    "Shasta funding fee exceeds 2 TRX. Try again later",
    "Shasta 发放手续费超过 2 TRX，请稍后再试"
  ],
  "尚未設定此網路的測試幣發放帳戶，請使用外部水龍頭": [
    "No funding account is configured for this network. Use an external faucet",
    "尚未设置此网络的测试币发放账户，请使用外部水龙头"
  ],
  "此網路尚未提供這種測試幣": [
    "This test token is not available on this network",
    "此网络尚未提供这种测试币"
  ],
  "僅能領取至此專案已建立的本機帳戶": [
    "Claims can only be sent to local accounts created in this project",
    "仅能领取至此项目已创建的本机账户"
  ],
  "目前選用的是測試幣發放帳戶；請切換至你的測試錢包再領取": [
    "The funding account is selected. Switch to your test wallet before claiming",
    "目前选用的是测试币发放账户；请切换至你的测试钱包再领取"
  ],
  "此網路本小時已達發放上限，請稍後再試": [
    "Hourly funding limit reached for this network. Try again later",
    "此网络本小时已达发放上限，请稍后再试"
  ],
  "測試幣發放帳戶庫存不足，請先補充或使用外部水龍頭": [
    "Faucet inventory is insufficient. Refill the funding account or use an external faucet",
    "测试币发放账户库存不足，请先补充或使用外部水龙头"
  ],
  "無法核對測試幣發放手續費": [
    "Unable to verify funding fees",
    "无法核对测试币发放手续费"
  ],
  "目前預估手續費超過測試幣發放上限，請稍後再試": [
    "Estimated fees exceed the faucet limit. Try again later",
    "目前预估手续费超过测试币发放上限，请稍后再试"
  ],
  "Devnet 空投未能確認：可能限流、庫存不足或連線逾時。請先查詢餘額，稍後再試或使用外部水龍頭": [
    "Devnet airdrop is unconfirmed: rate limits, inventory or a timeout may be responsible. Check your balance before retrying or use an external faucet",
    "Devnet 空投未能确认：可能限流、库存不足或连接逾时。请先查找余额，稍后再试或使用外部水龙头"
  ],
  "Solana RPC 查詢失敗或逾時；目前結果未知": [
    "Solana RPC failed or timed out; result unknown",
    "Solana RPC 查找失败或逾时；目前结果未知"
  ],
  "Solana RPC URL 無效": [
    "Invalid Solana RPC URL",
    "Solana RPC URL 无效"
  ],
  "Solana 交易日誌損壞": [
    "Solana transaction journal is damaged",
    "Solana 交易日志损坏"
  ],
  "Solana 日誌簽名無效": [
    "Invalid signature in the Solana journal",
    "Solana 日志签名无效"
  ],
  "僅允許 Solana Devnet，RPC 網路不符": [
    "Only Solana Devnet is allowed. RPC network mismatch",
    "仅允许 Solana Devnet，RPC 网络不符"
  ],
  "Solana 地址無效": [
    "Invalid Solana address",
    "Solana 地址无效"
  ],
  "Solana 儲存故障，請檢查磁碟後重啟": [
    "Solana storage failure. Check the disk before restarting",
    "Solana 保存故障，请检查磁盘后重启"
  ],
  "請先更新 Solana 交易紀錄，等待前筆 finalized": [
    "Refresh Solana history and wait for the previous transaction to finalize",
    "请先更新 Solana 交易记录，等待前笔 finalized"
  ],
  "目前僅支援有效的一般 Solana 錢包收款地址": [
    "Only valid standard Solana wallet receiving addresses are supported",
    "目前仅支持有效的一般 Solana 钱包收款地址"
  ],
  "SOL 數量無效，最多 9 位小數": [
    "Invalid SOL amount; up to 9 decimal places",
    "SOL 数量无效，最多 9 位小数"
  ],
  "Solana 儲存故障": [
    "Solana storage failure",
    "Solana 保存故障"
  ],
  "簽名地址不符": [
    "Signer address mismatch",
    "签名地址不符"
  ],
  "原 blockhash 已到期；仍保留日誌，請先查核原簽名的鏈上結果": [
    "Original blockhash expired. Journal retained; verify the original signature on-chain first",
    "原 blockhash 已到期；仍保留日志，请先查核原签名的链上结果"
  ],
  "找不到原交易": [
    "Original transaction not found",
    "找不到原交易"
  ],
  "TRON RPC URL 無效": [
    "Invalid TRON RPC URL",
    "TRON RPC URL 无效"
  ],
  "TRON 日誌無法讀取": [
    "Unable to read the TRON journal",
    "TRON 日志无法读取"
  ],
  "TRON 日誌交易損壞": [
    "Damaged transaction in the TRON journal",
    "TRON 日志交易损坏"
  ],
  "TRON 日誌簽名不符": [
    "Signature mismatch in the TRON journal",
    "TRON 日志签名不符"
  ],
  "TRON 儲存故障，請檢查磁碟後重啟": [
    "TRON storage failure. Check the disk before restarting",
    "TRON 保存故障，请检查磁盘后重启"
  ],
  "TRON 不允許 TRX 轉給自己，請填入另一個 Shasta 收款地址": [
    "TRON does not allow TRX self-transfers. Enter another Shasta receiving address",
    "TRON 不允许 TRX 转给自己，请填入另一个 Shasta 收款地址"
  ],
  "請先從 Shasta 水龍頭領取 TRX，啟用帳戶並支付費用": [
    "Get TRX from a Shasta faucet to activate the account and pay fees",
    "请先从 Shasta 水龙头领取 TRX，激活账户并支付费用"
  ],
  "TRON 費率參數缺漏或超出本原型支援範圍": [
    "TRON fee parameters are missing or outside this prototype's supported range",
    "TRON 费率参数缺漏或超出本原型支持范围"
  ],
  "轉帳數量必須大於零": [
    "Transfer amount must be greater than zero",
    "转账数量必须大于零"
  ],
  "TRX 金額超出範圍": [
    "TRX amount is out of range",
    "TRX 金额超出范围"
  ],
  "Energy 費用預留超過本原型每筆 100 TRX 限制": [
    "Energy reserve exceeds this prototype's 100 TRX per-transaction limit",
    "Energy 费用预留超过本原型每笔 100 TRX 限制"
  ],
  "TRON 節點區塊時間不新鮮，請稍後重試": [
    "TRON node block time is stale. Try again later",
    "TRON 节点区块时间不新鲜，请稍后重试"
  ],
  "簽名帳戶與報價不符": [
    "Signing account does not match the quote",
    "签名账户与报价不符"
  ],
  "原交易已過期；保留紀錄並查核收據，請勿盲目重送新交易": [
    "Original transaction expired. Keep the record and verify receipts before creating another transaction",
    "原交易已过期；保留记录并查核收据，请勿盲目重送新交易"
  ],
  "TRON 交易 ID 格式錯誤": [
    "Invalid TRON transaction ID",
    "TRON 交易 ID 格式错误"
  ],
  "TRON 儲存故障": [
    "TRON storage failure",
    "TRON 保存故障"
  ],
  "部分交易未能完成鏈上查核；以下保留本機最後紀錄，請稍後更新。": [
    "Some on-chain checks could not finish. The last local records are retained below; refresh later.",
    "部分交易未能完成链上查核；以下保留本机最后记录，请稍后更新。"
  ],
  "Devnet 空投限流，請稍後再試": [
    "Devnet airdrop rate limit reached. Try again later",
    "Devnet 空投限流，请稍后再试"
  ],
  "Shasta 測試幣庫存不足": [
    "Insufficient Shasta faucet inventory",
    "Shasta 测试币库存不足"
  ],
  "手續費另以 ETH 支付。": [
    "Fees are paid separately in ETH.",
    "手续费另以 ETH 支付。"
  ],
  "手續費另以 POL 支付。": [
    "Fees are paid separately in POL.",
    "手续费另以 POL 支付。"
  ],
  "查詢目前網路任意地址的 ETH 餘額。": [
    "Check the ETH balance of any address on this network.",
    "查找目前网络任意地址的 ETH 余额。"
  ],
  "查詢目前網路任意地址的 POL 餘額。": [
    "Check the POL balance of any address on this network.",
    "查找目前网络任意地址的 POL 余额。"
  ],
  "貼上公開地址，即可查看 ETH 餘額。": [
    "Paste a public address to view its ETH balance.",
    "粘贴公开地址，即可查看 ETH 余额。"
  ],
  "貼上公開地址，即可查看 POL 餘額。": [
    "Paste a public address to view its POL balance.",
    "粘贴公开地址，即可查看 POL 余额。"
  ],
  "帳戶 ${number}": [
    "Account ${number}",
    "账户 ${number}"
  ],
  "等待 ${hash} 的鏈上收據": [
    "Waiting for the on-chain receipt of ${hash}",
    "等待 ${hash} 的链上收据"
  ],
  "查詢 Slot ${slot}": [
    "Queried slot ${slot}",
    "查找 Slot ${slot}"
  ],
  "實際費用 ${fee}": [
    "Actual fee ${fee}",
    "实际费用 ${fee}"
  ],
  "執行結果 ${result}": [
    "Execution result ${result}",
    "运行结果 ${result}"
  ],
  "代幣查詢失敗：${error}": [
    "Token query failed: ${error}",
    "代币查找失败：${error}"
  ],
  "無法更新證據：${error}": [
    "Unable to refresh evidence: ${error}",
    "无法更新证据：${error}"
  ],
  "被授權地址 ${address}": [
    "Spender ${address}",
    "被授权地址 ${address}"
  ],
  "收款人 ${address}": [
    "Recipient ${address}",
    "收款人 ${address}"
  ],
  "已收錄的替代交易：${hash}": [
    "Included replacement: ${hash}",
    "已收录的替代交易：${hash}"
  ],
  "此網路尚未配置兌換合約": [
    "Exchange contracts are not configured on this network",
    "此网络尚未配置兑换合约"
  ],
  "兌換資產不可相同": [
    "Swap assets must be different",
    "兑换资产不可相同"
  ],
  "兌換數量必須大於 0": [
    "Swap amount must be greater than 0",
    "兑换数量必须大于 0"
  ],
  "此池無可用報價": [
    "No quote available for this pool",
    "此池无可用报价"
  ],
  "最多支援 20 個本機帳戶": [
    "Up to 20 local accounts are supported",
    "最多支持 20 个本机账户"
  ],
  "帳戶 ID 無效": [
    "Invalid account ID",
    "账户 ID 无效"
  ],
  "找不到帳戶": [
    "Account not found",
    "找不到账户"
  ],
  "帳戶尚未完成登記": [
    "Account registration is incomplete",
    "账户尚未完成登记"
  ],
  "無法解析交易日誌檔": [
    "Unable to parse the transaction journal",
    "无法解析交易日志档"
  ],
  "交易日誌包含空紀錄": [
    "Transaction journal contains an empty record",
    "交易日志包含空记录"
  ],
  "交易日誌已損毀": [
    "Transaction journal is damaged",
    "交易日志已损毁"
  ],
  "交易日誌的簽名資料不符": [
    "Transaction journal signature data does not match",
    "交易日志的签名数据不符"
  ],
  "找不到欲更新的交易紀錄": [
    "Transaction record to update was not found",
    "找不到欲更新的交易记录"
  ],
  "封存日誌格式錯誤": [
    "Invalid archive format",
    "封存日志格式错误"
  ],
  "封存含未確定紀錄": [
    "Archive contains unfinalized records",
    "封存含未确定记录"
  ],
  "封存簽名資料不符": [
    "Archive signature data does not match",
    "封存签名数据不符"
  ],
  "同步進度檔格式錯誤": [
    "Invalid sync progress file",
    "同步进度档格式错误"
  ],
  "錢包資料夾正由另一個程序使用": [
    "Another process is using this wallet directory",
    "钱包文件夹正由另一个进程使用"
  ],
  "交易儲存發生錯誤，請修復磁碟後重啟錢包": [
    "Transaction storage failed. Repair the disk before restarting the wallet",
    "交易保存发生错误，请修复磁盘后重启钱包"
  ],
  "此網路支援原生幣與 ERC-20 收付款；兌換目前只配置 Ethereum Sepolia": [
    "This network supports native and ERC-20 transfers. Exchange is configured only on Ethereum Sepolia",
    "此网络支持原生币与 ERC-20 收付款；兑换目前只配置 Ethereum Sepolia"
  ],
  "金鑰地址與報價不符": [
    "Key address does not match the quote",
    "密钥地址与报价不符"
  ],
  "代幣餘額不足以支付轉帳金額": [
    "Insufficient token balance for the transfer",
    "代币余额不足以支付转账金额"
  ],
  "目前基本費用超過報價上限，請重新預估": [
    "Current base fee exceeds the quote cap. Request a new quote",
    "目前基本费用超过报价上限，请重新预估"
  ],
  "代幣資料已變更，請重新預估": [
    "Token metadata changed. Request a new quote",
    "代币数据已变更，请重新预估"
  ],
  "交易需要更多 Gas，請重新預估": [
    "The transaction needs more gas. Request a new quote",
    "交易需要更多 Gas，请重新预估"
  ],
  "儲存的交易雜湊不符": [
    "Stored transaction hash does not match",
    "保存的交易哈希不符"
  ],
  "金額格式不正確，僅支援無正負號的十進位數字，不可包含科學記號或非數字字元": [
    "Invalid amount. Use an unsigned decimal number without scientific notation or other characters",
    "金额格式不正确，仅支持无正负号的十进位数字，不可包含科学记号或非数字字符"
  ],
  "金額小數位數超出該資產支援的上限": [
    "Amount exceeds this asset's supported decimal precision",
    "金额小数字数超出该资产支持的上限"
  ],
  "金額數值解析失敗": [
    "Unable to parse the amount",
    "金额数值解析失败"
  ],
  "金額超出 uint256 上限": [
    "Amount exceeds the uint256 limit",
    "金额超出 uint256 上限"
  ],
  "TRON Shasta RPC 無法完成查核；結果未知，請稍後更新": [
    "TRON Shasta RPC verification failed; result unknown. Refresh later",
    "TRON Shasta RPC 无法完成查核；结果未知，请稍后更新"
  ],
  "請輸入完整的 TRON Base58Check 地址": [
    "Enter a complete TRON Base58Check address",
    "请输入完整的 TRON Base58Check 地址"
  ],
  "TRON 地址格式錯誤": [
    "Invalid TRON address format",
    "TRON 地址格式错误"
  ],
  "TRON 地址校驗失敗或為零地址": [
    "Invalid TRON checksum or zero address",
    "TRON 地址校验失败或为零地址"
  ],
  "RPC 並非 TRON Shasta，已停止操作": [
    "RPC is not TRON Shasta. Operation stopped",
    "RPC 并非 TRON Shasta，已停止操作"
  ],
  "兌換只支援 Sepolia 官方 WETH 與 Circle 測試 USDC": [
    "Swaps support only Sepolia WETH and Circle test USDC",
    "兑换只支持 Sepolia 官方 WETH 与 Circle 测试 USDC"
  ],
  "包裝與解包只能使用指定的 Sepolia WETH": [
    "Wrapping and unwrapping require the configured Sepolia WETH",
    "包装与解包只能使用指定的 Sepolia WETH"
  ],
  "包裝與解包不接受交易池或滑價參數": [
    "Pool and slippage parameters are not allowed for wrapping or unwrapping",
    "包装与解包不接受交易池或滑价参数"
  ],
  "WETH 餘額不足": [
    "Insufficient WETH balance",
    "WETH 余额不足"
  ],
  "兌換的兩種資產不可相同": [
    "Swap assets must be different",
    "兑换的两种资产不可相同"
  ],
  "滑價必須介於 0.01% 至 5%": [
    "Slippage must be between 0.01% and 5%",
    "滑价必须介于 0.01% 至 5%"
  ],
  "請選擇有效的 Uniswap V3 費率": [
    "Select a valid Uniswap V3 fee tier",
    "请选择有效的 Uniswap V3 费率"
  ],
  "兌換來源代幣餘額不足": [
    "Insufficient input token balance",
    "兑换来源代币余额不足"
  ],
  "請先授權 Router 本次兌換所需數量，等待授權交易成功，再重新報價": [
    "Approve the Router for this swap amount and wait for success before requesting a new quote",
    "请先授权 Router 本次兑换所需数量，等待授权交易成功，再重新报价"
  ],
  "可收到的數量太小或交易池沒有足夠流動性": [
    "Output is too small or the pool has insufficient liquidity",
    "可收到的数量太小或交易池没有足够流动性"
  ],
  "未知的兌換操作": [
    "Unknown exchange action",
    "未知的兑换操作"
  ],
  "兌換報價資料不完整": [
    "Incomplete swap quote data",
    "兑换报价数据不完整"
  ],
  "兌換來源代幣餘額已不足": [
    "Input token balance is no longer sufficient",
    "兑换来源代币余额已不足"
  ],
  "兌換所需授權額度已不足": [
    "Swap allowance is no longer sufficient",
    "兑换所需授权额度已不足"
  ],
  "無法解析交易池地址": [
    "Unable to parse the pool address",
    "无法解析交易池地址"
  ],
  "此費率沒有 WETH/USDC 交易池，請選擇其他費率": [
    "No WETH/USDC pool exists at this fee tier. Select another tier",
    "此费率没有 WETH/USDC 交易池，请选择其他费率"
  ],
  "鏈上報價失敗；RPC 或此交易池的流動性目前無法完成兌換": [
    "On-chain quote failed. RPC or pool liquidity currently prevents the swap",
    "链上报价失败；RPC 或此交易池的流动性目前无法完成兑换"
  ],
  "兌換報價格式錯誤": [
    "Invalid swap quote format",
    "兑换报价格式错误"
  ],
  "收支索引檔格式錯誤": [
    "Invalid activity index file",
    "收支索引档格式错误"
  ],
  "收支索引檔雜湊錯誤": [
    "Invalid hash in the activity index",
    "收支索引档哈希错误"
  ],
  "交易尚未取得有效的鏈上收據，請稍後再匯入": [
    "A valid on-chain receipt is not available yet. Try importing later",
    "交易尚未取得有效的链上收据，请稍后再导入"
  ],
  "每次同步最多 20 個代幣合約": [
    "Up to 20 token contracts per sync",
    "每次同步最多 20 个代币合约"
  ],
  "頁碼必須大於 0": [
    "Page number must be greater than 0",
    "页码必须大于 0"
  ],
  "頁碼超出範圍": [
    "Page number is out of range",
    "页码超出范围"
  ],
  "收支金額格式錯誤": [
    "Invalid activity amount",
    "收支金额格式错误"
  ],
  "兌換資產只能回到自己的錢包": [
    "Swap output must return to your own wallet",
    "兑换资产只能回到自己的钱包"
  ],
  "轉帳金額必須大於 0": [
    "Transfer amount must be greater than 0",
    "转账金额必须大于 0"
  ],
  "代幣轉帳必須指定 contract 合約地址": [
    "Token transfer requires a contract address",
    "代币转账必须指定 contract 合约地址"
  ],
  "轉帳代幣數量必須大於 0": [
    "Token transfer amount must be greater than 0",
    "转账代币数量必须大于 0"
  ],
  "代幣餘額不足": [
    "Insufficient token balance",
    "代币余额不足"
  ],
  "calldata 驗證失敗": [
    "Calldata validation failed",
    "calldata 验证失败"
  ],
  "代幣授權必須指定 contract 合約地址": [
    "Token approval requires a contract address",
    "代币授权必须指定 contract 合约地址"
  ],
  "授權數量不可為負數": [
    "Allowance cannot be negative",
    "授权数量不可为负数"
  ],
  "不支援的 action 操作，僅允許 eth、transfer、approve、wrap、unwrap 或 swap": [
    "Unsupported action. Allowed: eth, transfer, approve, wrap, unwrap or swap",
    "不支持的 action 操作，仅允许 eth、transfer、approve、wrap、unwrap 或 swap"
  ],
  "無法取得 EIP-1559 基本費用，請稍後重試": [
    "EIP-1559 base fee unavailable. Try again later",
    "无法取得 EIP-1559 基本费用，请稍后重试"
  ],
  "無法取得優先費用": [
    "Priority fee unavailable",
    "无法取得优先费用"
  ],
  "Gas 預估值無效": [
    "Invalid gas estimate",
    "Gas 预估值无效"
  ],
  "L1／營運費已超過預留估算，請重新報價": [
    "L1 / operator fees exceed the reserve. Request a new quote",
    "L1／营运费已超过预留估算，请重新报价"
  ],
  "此 Nonce 已有收錄交易，請先更新紀錄": [
    "A transaction at this nonce is already included. Refresh history first",
    "此 Nonce 已有收录交易，请先更新记录"
  ],
  "無法替代這筆交易": [
    "This transaction cannot be replaced",
    "无法替代这笔交易"
  ],
  "原交易簽名地址不符": [
    "Original transaction signer does not match",
    "原交易签名地址不符"
  ],
  "無法取得基本費用": [
    "Base fee unavailable",
    "无法取得基本费用"
  ],
  "交易日誌格式錯誤": [
    "Invalid transaction journal format",
    "交易日志格式错误"
  ],
  "原交易金額格式錯誤": [
    "Invalid original transaction amount",
    "原交易金额格式错误"
  ],
  "Solana 備份格式無效": [
    "Invalid Solana backup format",
    "Solana 备份格式无效"
  ],
  "Solana 備份地址無效": [
    "Invalid Solana backup address",
    "Solana 备份地址无效"
  ],
  "Solana 備份資料損壞": [
    "Damaged Solana backup data",
    "Solana 备份数据损坏"
  ],
  "Solana 備份地址不符": [
    "Solana backup address mismatch",
    "Solana 备份地址不符"
  ],
  "無法讀取既有金鑰檔": [
    "Unable to read the existing key file",
    "无法读取既有密钥档"
  ],
  "金鑰檔地址格式錯誤": [
    "Invalid key file address format",
    "密钥档地址格式错误"
  ],
  "無法安全讀取錢包儲存狀態": [
    "Unable to safely read wallet storage state",
    "无法安全读取钱包保存状态"
  ],
  "請選擇 V3 scrypt 加密的 Keystore JSON": [
    "Choose a V3 scrypt-encrypted Keystore JSON file",
    "请选择 V3 scrypt 加密的 Keystore JSON"
  ],
  "Keystore 密碼運算參數無效": [
    "Invalid Keystore password-processing parameters",
    "Keystore 密码运算参数无效"
  ],
  "Keystore 密碼運算參數不在支援範圍": [
    "Keystore password-processing parameters are outside supported bounds",
    "Keystore 密码运算参数不在支持范围"
  ],
  "Keystore salt 格式錯誤": [
    "Invalid Keystore salt format",
    "Keystore salt 格式错误"
  ],
  "Keystore 加密欄位格式錯誤": [
    "Invalid Keystore encryption fields",
    "Keystore 加密字段格式错误"
  ],
  "既有帳戶資料無法讀取": [
    "Unable to read existing account data",
    "既有账户数据无法读取"
  ],
  "此地址已存在另一個帳戶，請切換既有帳戶": [
    "This address already exists in another account. Switch to that account",
    "此地址已存在另一个账户，请切换既有账户"
  ],
  "無法解析代幣 symbol 返回值": [
    "Unable to decode the token symbol response",
    "无法解析代币 symbol 返回值"
  ],
  "無法解析代幣 decimals 返回值": [
    "Unable to decode the token decimals response",
    "无法解析代币 decimals 返回值"
  ],
  "無法解析代幣 balanceOf 返回值": [
    "Unable to decode the token balanceOf response",
    "无法解析代币 balanceOf 返回值"
  ],
  "無法解析代幣 allowance 返回值": [
    "Unable to decode the token allowance response",
    "无法解析代币 allowance 返回值"
  ],
  "標準 transfer/approve calldata 必須為 68 位元組": [
    "Standard transfer/approve calldata must be 68 bytes",
    "标准 transfer/approve calldata 必须为 68 字节"
  ],
  "無法解析 transfer calldata": [
    "Unable to decode transfer calldata",
    "无法解析 transfer calldata"
  ],
  "transfer calldata 參數型別不符": [
    "Invalid transfer calldata parameter types",
    "transfer calldata 参数类型不符"
  ],
  "無法解析 approve calldata": [
    "Unable to decode approve calldata",
    "无法解析 approve calldata"
  ],
  "approve calldata 參數型別不符": [
    "Invalid approve calldata parameter types",
    "approve calldata 参数类型不符"
  ],
  "未知的 ERC20 方法選擇器": [
    "Unknown ERC-20 method selector",
    "未知的 ERC20 方法选择器"
  ],
  "指定的單一 finalized 區塊；點選交易再核對收據": [
    "One specified finalized block. Select a transaction to verify its receipt",
    "指定的单一 finalized 区块；点击交易再核对收据"
  ],
  "Content-Type 必須是 application/json": [
    "Content-Type must be application/json",
    "Content-Type 必须是 application/json"
  ],
  "請求資料格式錯誤或含有未定義欄位": [
    "Invalid request format or unknown fields",
    "请求数据格式错误或含有未定义字段"
  ],
  "請求結尾含有多餘資料": [
    "Unexpected data after the request body",
    "请求结尾含有多余数据"
  ],
  "頁碼格式錯誤": [
    "Invalid page number format",
    "页码格式错误"
  ],
  "本機錢包儲存失敗，請檢查資料磁碟與權限": [
    "Local wallet storage failed. Check the disk and permissions",
    "本机钱包保存失败，请检查数据磁盘与权限"
  ],
  "無效的 Host 標頭，僅允許本機存取": [
    "Invalid Host header. Only local access is allowed",
    "无效的 Host 标头，仅允许本机访问"
  ],
  "跨來源請求已被拒絕": [
    "Cross-origin request rejected",
    "跨来源请求已被拒绝"
  ],
  "缺少或無效的 CSRF Token (X-Wallet-CSRF)": [
    "Missing or invalid CSRF token (X-Wallet-CSRF)",
    "缺少或无效的 CSRF Token (X-Wallet-CSRF)"
  ],
  "請至少等待一分鐘再申請 Devnet 空投": [
    "Wait at least one minute before requesting another Devnet airdrop",
    "请至少等待一分钟再申请 Devnet 空投"
  ],
  "此交易沒有可辨識、與本錢包相關的收支": [
    "No recognizable activity involving this wallet was found in this transaction",
    "此交易没有可辨识、与本钱包相关的收支"
  ],
  "起始區塊超過最新區塊": [
    "Start block is newer than the latest block",
    "起始区块超过最新区块"
  ],
  "此範圍相關交易過多，請用交易雜湊個別匯入": [
    "Too many relevant transactions in this range. Import individual transaction hashes instead",
    "此范围相关交易过多，请用交易哈希个别导入"
  ],
  "交易雜湊格式不正確，請輸入 0x 開頭的 64 位十六進位雜湊": [
    "Invalid transaction hash. Enter 0x followed by 64 hexadecimal characters",
    "交易哈希格式不正确，请输入 0x 开头的 64 位十六进位哈希"
  ],
  "RPC 連到其他網路，已停止操作；RPC 必須符合目前選擇的測試網": [
    "RPC network mismatch. Operation stopped; RPC must match the selected testnet",
    "RPC 连到其他网络，已停止操作；RPC 必须符合目前选择的测试网"
  ],
  "暫時無法取得目前測試網資料，請稍後重試": [
    "Testnet data is temporarily unavailable. Try again later",
    "暂时无法取得目前测试网数据，请稍后重试"
  ],
  "測試網查詢逾時，結果未知，請稍後重試": [
    "Testnet query timed out; result unknown. Try again later",
    "测试网查找逾时，结果未知，请稍后重试"
  ],
  "此 RPC 尚未找到這筆交易，請確認網路與雜湊，或稍後重試": [
    "This RPC has not found the transaction. Check the network and hash, or retry later",
    "此 RPC 尚未找到这笔交易，请确认网络与哈希，或稍后重试"
  ],
  "SEPOLIA_RPC_URL 必須是 HTTP 或 HTTPS RPC 網址": [
    "SEPOLIA_RPC_URL must be an HTTP or HTTPS RPC URL",
    "SEPOLIA_RPC_URL 必须是 HTTP 或 HTTPS RPC 网址"
  ],
  "更新於 ${time}": [
    "Updated ${time}",
    "更新于 ${time}"
  ],
  "先到「領取測試幣」補充 ETH 與 USDC；本機庫存不足時，可使用下方外部水龍頭。": [
    "Open Get test tokens to top up ETH and USDC. Use an external faucet below if local inventory is empty.",
    "先到「领取测试币」补充 ETH 与 USDC；本机库存不足时，可使用下方外部水龙头。"
  ],
  "前往領取測試幣 →": [
    "Get test tokens →",
    "前往领取测试币 →"
  ],
  "Polygon Amoy：尚缺測試 POL 與真實轉帳驗收。": [
    "Polygon Amoy: test POL funding and a real transfer acceptance are still pending.",
    "Polygon Amoy：尚缺测试 POL 与真实转账验收。"
  ],
  "Solana Devnet：公開水龍頭回覆 429，其他水龍頭的 GitHub 登入授權仍待確認。": [
    "Solana Devnet: the public faucet returned 429; GitHub sign-in approval for other faucets is pending.",
    "Solana Devnet：公开水龙头回复 429，其他水龙头的 GitHub 登录授权仍待确认。"
  ],
  "Ethereum Sepolia 以外的 DEX 與 Solana SPL 代幣轉帳尚未實作。": [
    "DEX support outside Ethereum Sepolia and Solana SPL transfers are not implemented.",
    "Ethereum Sepolia 以外的 DEX 与 Solana SPL 代币转账尚未实作。"
  ],
  "終局確認": [
    "Finalized",
    "最终确认"
  ],
  "活動": [
    "Activity",
    "活动"
  ],
  "收支明細": [
    "Asset movements",
    "收支明细"
  ],
  "交易狀態": [
    "Transaction status",
    "交易状态"
  ],
  "重新整理": [
    "Refresh",
    "刷新"
  ],
  "更多操作": [
    "More options",
    "更多操作"
  ],
  "查詢其他交易": [
    "Look up another transaction",
    "查询其他交易"
  ],
  "開發者：同步與匯入": [
    "Developer: sync and import",
    "开发者：同步与导入"
  ],
  "活動檢視": [
    "Activity views",
    "活动视图"
  ],
  "查看交易狀態，或切換收支明細核對資產變動。": [
    "Review transaction status or switch to asset movements.",
    "查看交易状态，或切换收支明细核对资产变动。"
  ],
  "最後更新：${time(new Date().toISOString())}": [
    "Last updated: ${time(new Date().toISOString())}",
    "最后更新：${time(new Date().toISOString())}"
  ],
  "尚無已收錄的收支；未收錄不代表沒有鏈上交易。可至區塊瀏覽器核對。": [
    "No recorded movements. Unrecorded transactions may exist; check the block explorer.",
    "暂无已收录的收支；未收录不代表没有链上交易。可至区块浏览器核对。"
  ],
  "搜尋我的錢包或常用地址": [
    "Search my wallets or contacts",
    "搜索我的钱包或常用地址"
  ],
  "搜尋常用地址": [
    "Search contacts",
    "搜索常用地址"
  ],
  "輸入名稱或地址": [
    "Enter a name or address",
    "输入名称或地址"
  ],
  "儲存此收款地址": [
    "Save this recipient",
    "保存此收款地址"
  ],
  "復原移除": [
    "Undo removal",
    "撤销移除"
  ],
  "常用地址": [
    "Contacts",
    "常用地址"
  ],
  "我的錢包": [
    "My wallets",
    "我的钱包"
  ],
  "編輯名稱": [
    "Edit name",
    "编辑名称"
  ],
  "已複製地址。": [
    "Address copied.",
    "已复制地址。"
  ],
  "已移除，可復原。": [
    "Removed. You can undo this.",
    "已移除，可撤销。"
  ],
  "已復原。": [
    "Restored.",
    "已恢复。"
  ],
  "尚無符合的常用地址。": [
    "No matching contacts.",
    "暂无符合的常用地址。"
  ],
  "常用地址僅儲存在此瀏覽器，依測試網路分開；不會跨裝置同步。轉帳前請核對完整地址。": [
    "Contacts are stored only in this browser, separately for each test network. They do not sync across devices. Verify the full address before sending.",
    "常用地址仅保存在此浏览器，依测试网络分开；不会跨设备同步。转账前请核对完整地址。"
  ],
  "操作未完成，請確認瀏覽器權限後重試。": [
    "Could not complete the action. Check browser permissions and try again.",
    "操作未完成，请确认浏览器权限后重试。"
  ],
  "請輸入目前網路的完整有效地址；送出交易前仍會查核。": [
    "Enter a complete address for this network; it will also be validated before sending.",
    "请输入当前网络的完整有效地址；发送交易前仍会核验。"
  ],
  "此地址已有新資料，未覆寫。": [
    "This address has newer data; it was not overwritten.",
    "此地址已有新数据，未覆盖。"
  ],
  "觀察公開地址": [
    "Watch a public address",
    "观察公开地址"
  ],
  "說明與診斷": [
    "Help and diagnostics",
    "帮助与诊断"
  ],
  "開發者診斷": [
    "Developer diagnostics",
    "开发者诊断"
  ],
  "進階：包裝與直接兌換": [
    "Advanced: wrapping and direct swaps",
    "进阶：包装与直接兑换"
  ],
  "完整收款地址": [
    "Full recipient address",
    "完整收款地址"
  ],
  "請使用符合目前網路的外部水龍頭；領取後重新整理餘額。": [
    "Use an external faucet for this network, then refresh your balance.",
    "请使用符合当前网络的外部水龙头；领取后刷新余额。"
  ],
  "外部服務可能要求登入或符合領取資格；請勿提供助記詞或私鑰。": [
    "External services may require sign-in or eligibility checks. Never provide your recovery phrase or private key.",
    "外部服务可能要求登录或符合领取资格；请勿提供助记词或私钥。"
  ]
};

Object.assign(window.FlowMessages, {
  '合約功能':['Contract features','合约功能'],
  '付款託管':['Payment escrow','付款托管'],
  "練習存入、取回測試 ETH，或用測試 USDC 付款。":["Practice depositing and withdrawing test ETH, or paying with test USDC.", "练习存入、取回测试 ETH，或用测试 USDC 付款。"],
  '測試 USDC 付款託管':['Test USDC payment escrow','测试 USDC 付款托管'],
  "款項先由合約保管。付款人可以放款給收款人，收款人可以退款給原付款人。":["The contract holds the funds. The payer can release them to the recipient, or the recipient can refund the original payer.", "款项先由合约保管。付款人可以放款给收款人，收款人可以退款给原付款人。"],
  '付款流程':['Payment steps','付款流程'],
  '授權金額':['Approve amount','授权金额'],
  '付款至合約':['Pay into escrow','付款至合约'],
  '放款或退款':['Release or refund','放款或退款'],
  '放款給收款人':['Release to recipient','放款给收款人'],
  '退款給付款人':['Refund to payer','退款给付款人'],
  '正在讀取付款託管狀態…':['Loading payment escrow…','正在读取付款托管状态…'],
  '正在更新付款託管狀態…':['Updating payment escrow…','正在更新付款托管状态…'],
  '更新付款狀態':['Refresh payment status','更新付款状态'],
  "雙方都不操作，款項就會留在合約。本功能不會自動退款，也不處理交易糾紛。":["If neither person acts, the funds stay in the contract. There are no automatic refunds or dispute resolution.", "双方都不操作，款项就会留在合约。本功能不会自动退款，也不处理交易纠纷。"],
  "這是共用錢包，其他人也能操作款項。請用兩個不同錢包測試付款和收款。":["Other people can use this shared wallet too. Use two different wallets to test paying and receiving.", "这是共用钱包，其他人也能操作款项。请用两个不同钱包测试付款和收款。"],
  '建立付款':['Create payment','创建付款'],
  '訂單編號':['Order reference','订单编号'],
  '例如 order-001':['e.g. order-001','例如 order-001'],
  "限英文字母、數字、- 或 _，最多 64 字元。收款人查詢時需要編號和付款人地址。":["Use up to 64 letters, digits, - or _. The recipient needs this reference and the payer address to find the order.", "限英文字母、数字、- 或 _，最多 64 字符。收款人查询时需要编号和付款人地址。"],
  '收款人地址':['Recipient address','收款人地址'],
  '付款金額（測試 USDC）':['Payment amount (test USDC)','付款金额（测试 USDC）'],
  '例如 5':['e.g. 5','例如 5'],
  '核對付款資料':['Review payment','核对付款资料'],
  "授權只設定可扣款金額，不會付款。每一步都要確認金額、手續費和密碼。":["Approval only sets how much the contract can use; it does not pay. Review the amount and fee, then enter your password for each step.", "授权只设置可扣款金额，不会付款。每一步都要确认金额、手续费和密码。"],
  '查詢訂單':['Find order','查询订单'],
  '原付款人地址':['Original payer address','原付款人地址'],
  '要查詢的訂單編號':['Order reference to find','要查询的订单编号'],
  '核對並放款':['Review release','核对并放款'],
  '核對並全額退款':['Review full refund','核对并全额退款'],
  "付款紀錄":["Payment activity", "付款记录"],
  '尚未付款':['Not paid','尚未付款'],
  "款項由合約保管":["Funds held by the contract", "款项由合约保管"],
  '已放款給收款人':['Released to recipient','已放款给收款人'],
  '已退回付款人':['Refunded to payer','已退回付款人'],
  '付款金額請輸入最多 6 位小數的正數':['Enter a positive amount with up to 6 decimal places','付款金额请输入最多 6 位小数的正数'],
  '完成授權或付款後，交易紀錄會顯示在這裡。':['Approval and payment transactions will appear here.','完成授权或付款后，交易记录会显示在这里。'],
  "付款託管尚未開放。":["Payment escrow is not available yet.", "付款托管尚未开放。"],
  '可用餘額：${balance} USDC · 已授權：${allowance} USDC':['Available: ${balance} USDC · Approved: ${allowance} USDC','可用余额：${balance} USDC · 已授权：${allowance} USDC'],
  '暫時無法讀取付款狀態，請重試。':['Payment status is unavailable. Please retry.','暂时无法读取付款状态，请重试。'],
  '原付款人':['Original payer','原付款人'],
  '付款人':['Payer','付款人'],
  '收款人':['Recipient','收款人'],
  '金額':['Amount','金额'],
  '鏈上確認':['Chain confirmation','链上确认'],
  '已最終確認':['Finalized','已最终确认'],
  '已收錄，仍待最終確認':['Included, awaiting finality','已收录，仍待最终确认'],
  "找不到已付款的訂單，請確認付款人地址和訂單編號。":["No paid order found. Check the payer address and order reference.", "找不到已付款的订单，请确认付款人地址和订单编号。"],
  "你是付款人。確認對方已完成約定，再放款給收款人。退款要由收款人操作。":["You are the payer. Release the funds when the recipient has done what you agreed. Only the recipient can refund you.", "你是付款人。确认对方已完成约定，再放款给收款人。退款要由收款人操作。"],
  "你是收款人。可將全額退給原付款人。要收款，請等付款人放款。":["You are the recipient. You can refund the full amount to the original payer. To receive the funds, wait for the payer to release them.", "你是收款人。可将全额退给原付款人。要收款，请等付款人放款。"],
  "這個錢包只能查看。請切換到付款人或收款人的錢包。":["This wallet can only view the order. Switch to the payer or recipient wallet to act.", "这个钱包只能查看。请切换到付款人或收款人的钱包。"],
  "這筆訂單已完成，不能再放款或退款。":["This order is complete. It cannot be released or refunded again.", "这笔订单已完成，不能再放款或退款。"],
  '收款人須為另一個錢包地址':['The recipient must be a different wallet address','收款人须为另一个钱包地址'],
  '測試 USDC 餘額不足，請先領取測試幣。':['Insufficient test USDC. Get test tokens first.','测试 USDC 余额不足，请先领取测试币。'],
  '此訂單編號已付款，請查詢原訂單。':['This reference has already been paid. Look up the existing order.','此订单编号已付款，请查询原订单。'],
  "請先查詢款項仍由合約保管的訂單。":["First, find an order whose funds are still held by the contract.", "请先查询款项仍由合约保管的订单。"],
  '確認付款至合約':['Confirm payment into escrow','确认付款至合约'],
  '確認放款給收款人':['Confirm release to recipient','确认放款给收款人'],
  '確認全額退款':['Confirm full refund','确认全额退款'],
  '確認加速原交易':['Confirm speeding up the original transaction','确认加速原交易'],
  '原操作':['Original action','原操作'],
  '最多扣除 ${symbol}':['Maximum debit ${symbol}','最多扣除 ${symbol}'],
  '託管合約':['Escrow contract','托管合约'],
  '資金去向':['Funds destination','资金去向'],
  '目前錢包 → 託管合約':['Current wallet → Escrow contract','当前钱包 → 托管合约'],
  '託管合約 → 收款人':['Escrow contract → Recipient','托管合约 → 收款人'],
  '託管合約 → 原付款人':['Escrow contract → Original payer','托管合约 → 原付款人'],
  '操作說明':['What happens next','操作说明'],
  "款項會留在合約，等你放款給收款人，或由收款人退款。":["The contract will hold the funds until you release them or the recipient refunds you.", "款项会留在合约，等你放款给收款人，或由收款人退款。"],
  "交易成功後會轉出全部款項，不能再放款或退款。":["Once this transaction succeeds, the full amount is sent. The order cannot be released or refunded again.", "交易成功后会转出全部款项，不能再放款或退款。"],
  '先撤銷舊授權':['Revoke previous allowance first','先撤销旧授权'],
  '確認本次 USDC 授權':['Confirm USDC allowance','确认本次 USDC 授权'],
  "先取消舊的授權，再授權這次要付的金額。":["Cancel the old allowance first, then approve the amount you want to pay.", "先取消旧的授权，再授权这次要付的金额。"],
  "這一步只授權，不會付款。授權成功後，再按「核對付款資料」。":["This step only approves the amount; it does not pay. After approval succeeds, select Review payment again.", "这一步只授权，不会付款。授权成功后，再按“核对付款资料”。"],
  "授權已送出。等紀錄顯示成功，再按「核對付款資料」付款。":["Approval submitted. Once it succeeds in the activity list, select Review payment to pay.", "授权已发送。等记录显示成功，再按“核对付款资料”付款。"],
  "交易已送出，請查看訂單狀態和付款紀錄。":["Transaction submitted. Check the order status and payment activity.", "交易已发送，请查看订单状态和付款记录。"],
  '訂單編號請使用 1–64 個英文字母、數字、連字號或底線':['Use 1–64 letters, digits, hyphens or underscores for the order reference','订单编号请使用 1–64 个英文字母、数字、连字符或下划线'],
  '此環境尚未開放付款託管':['Payment escrow is not enabled in this environment','此环境尚未开放付款托管'],
  '託管合約與代幣由伺服器設定':['The server configures the escrow contract and token','托管合约与代币由服务器设置'],
  '此操作不接受訂單欄位':['This action does not accept order fields','此操作不接受订单字段'],
  '付款人必須是目前錢包':['The payer must be the current wallet','付款人必须是当前钱包'],
  '此付款人的訂單編號已使用，請查詢原訂單':['This payer has already used this reference. Look up the existing order','此付款人的订单编号已使用，请查询原订单'],
  '付款金額必須大於 0':['Payment amount must be greater than zero','付款金额必须大于 0'],
  '授權額度不足，請先完成本次金額的 USDC 授權':['Insufficient allowance. Approve this USDC amount first','授权额度不足，请先完成本次金额的 USDC 授权'],
  '訂單未在託管中，請更新狀態；已完成的訂單不能重複操作':['Refresh the order. Only funded, unsettled orders can be released or refunded','订单未在托管中，请更新状态；已完成的订单不能重复操作'],
  '只有付款人可以放款給收款人':['Only the payer can release funds to the recipient','只有付款人可以放款给收款人'],
  '只有收款人可以退款給原付款人':['Only the recipient can refund the original payer','只有收款人可以退款给原付款人'],
  '託管交易預先檢查未通過，請更新訂單、餘額與授權後重試；交易尚未送出':['Escrow preflight failed. Refresh the order, balance and allowance before retrying. Nothing was submitted','托管交易预先检查未通过，请更新订单、余额与授权后重试；交易尚未发送'],
  '託管設定已變更，請重新預估':['Escrow settings changed. Request a new quote','托管设置已变更，请重新预估'],
  '託管合約的付款代幣與設定不符':['The escrow payment token does not match the configured token','托管合约的付款代币与设置不符'],
  '託管僅支援設定的 6 位小數測試 USDC':['Escrow requires the configured test USDC with 6 decimals','托管仅支持设置的 6 位小数测试 USDC'],
  '無法讀取訂單狀態':['Unable to read the order state','无法读取订单状态'],
  '區塊已變更，請重新查詢訂單':['The block changed. Look up the order again','区块已变更，请重新查询订单']
});
