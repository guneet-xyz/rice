return {
  'folke/trouble.nvim',
  lazy = false,
  cmd = 'Trouble',
  config = function()
    vim.api.nvim_create_user_command('TroubleBuffer', function()
      vim.cmd 'Trouble diagnostics toggle filter = {buf = 0}'
    end, { desc = 'Open Trouble for current buffer' })
    require('trouble').setup {}
  end,
}
