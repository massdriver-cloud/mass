param md_metadata object
param params object

resource devtestlocal0000 'Microsoft.Storage/storageAccounts@2021-04-01' = {
  name: md_metadata.name_prefix
  kind: params.storage.type //'StorageV2'
  location: params.region //'EastUS'
  sku: {
    name: params.storage.sku //'PREMIUM_LRS'
  }
  tags: md_metadata.default_tags
}

output storageAccountId string = devtestlocal0000.id
