return {
  'qvalentin/helm-ls.nvim',
  config = function()
    local lspconfig = require 'lspconfig'
    -- setup helm-ls
    lspconfig.helm_ls.setup {
      settings = {
        ['helm-ls'] = {
          yamlls = {
            path = 'yaml-language-server',
          },
        },
      },
    }
  end,
}
