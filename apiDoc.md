GET /api/contractlist?username=     return [contractid1,contractid2] []string 	获取合约列表
GET /api/contract/id?username= 		return SignatureContract 					获取单个合约信息
GET /api/accout?username=			return Accout 								获取账户信息
POST /api/contract?username= 		return contractId string					广告主创建合约
GET /api/contractlist?username=		return [contractid1,contractid2] []string	获取已签约的合约列表
GET /api/confirmlist?username=		return [contractid1,contractid2] []string	获取未签约的合约列表
POST /api/sign?username=														媒体、第三方签名
POST /api/mediasubmit?username=													媒体推送log				{"ContractId":"xxx","FilePath":"ooo"}			
POST /api/signlog?username=														第三方确认log，并上传自己的判定结果								
GET /api/loglist?username=			return [logid1,logid2] []string				第三方获取媒体log列表
GET /api/log?username=				return MediaLogSubmit						第三方获取媒体log信息



confirm-contract 
