-- colorscheme plugins
local dependencies = {
  'folke/tokyonight.nvim',
  'catppuccin/nvim',
  'EdenEast/nightfox.nvim',
  'daschw/leaf.nvim',
  'rebelot/kanagawa.nvim',
  'rose-pine/neovim',
}

-- favorite color schemes
local colorscheme_filters = {
  '^.*fox$',
  'leaf',
  'retrobox',
  'tokyonight.*',
  'catppuccin.*',
  'kanagawa',
}

-- active colorscheme
local active_colorscheme = 'rose-pine'

-------------------------------------------------------
------- PLUGIN AND TELESCOPE DEFINITION FOLLOWS -------
-------------------------------------------------------

local actions = require 'telescope.actions'
local action_state = require 'telescope.actions.state'
local finders = require 'telescope.finders'
local pickers = require 'telescope.pickers'
local previewers = require 'telescope.previewers'
local utils = require 'telescope.utils'
local conf = require('telescope.config').values

-- snippet from https://github.com/nvim-telescope/telescope.nvim/blob/a4ed82509cecc56df1c7138920a1aeaf246c0ac5/lua/telescope/builtin/__internal.lua#L1009

return {
  'xiyaowong/transparent.nvim',
  dependencies = dependencies,
  priority = 1000, -- Make sure to load this before all the other start plugins.
  init = function()
    vim.api.nvim_create_user_command('BrowseColorschemes', function()
      local opts = require('telescope.themes').get_ivy {
        enable_preview = true,
        previewer = false,
        layout_config = {
          width = 0.5,
          height = 0.2,
        },
      }

      local before_background = vim.o.background
      local before_color = vim.api.nvim_exec2('colorscheme', { output = true }).output
      local need_restore = not not opts.enable_preview

      local colors = opts.colors or { before_color }
      if not vim.tbl_contains(colors, before_color) then
        table.insert(colors, 1, before_color)
      end

      colors = vim.list_extend(
        colors,
        vim.tbl_filter(function(color)
          return not vim.tbl_contains(colors, color)
        end, vim.fn.getcompletion('', 'color'))
      )

      -- if lazy is available, extend the colors list with unloaded colorschemes
      local lazy = package.loaded['lazy.core.util']
      if lazy and lazy.get_unloaded_rtp then
        local paths = lazy.get_unloaded_rtp ''
        local all_files = vim.fn.globpath(table.concat(paths, ','), 'colors/*', 1, 1)
        for _, f in ipairs(all_files) do
          local color = vim.fn.fnamemodify(f, ':t:r')
          if not vim.tbl_contains(colors, color) then
            table.insert(colors, color)
          end
        end
      end

      -- filter colorschemes
      if colorscheme_filters then
        colors = vim.tbl_filter(function(color)
          for _, filter in ipairs(colorscheme_filters) do
            if string.match(color, filter) then
              return true
            end
          end
          return false
        end, colors)
      end

      local previewer
      if opts.enable_preview then
        -- define previewer
        local bufnr = vim.api.nvim_get_current_buf()
        local p = vim.api.nvim_buf_get_name(bufnr)

        -- show current buffer content in previewer
        previewer = previewers.new_buffer_previewer {
          get_buffer_by_name = function()
            return p
          end,
          define_preview = function(self)
            if vim.loop.fs_stat(p) then
              conf.buffer_previewer_maker(p, self.state.bufnr, { bufname = self.state.bufname })
            else
              local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
              vim.api.nvim_buf_set_lines(self.state.bufnr, 0, -1, false, lines)
            end
          end,
        }
      end

      local picker = pickers.new(opts, {
        prompt_title = 'Change Colorscheme',
        finder = finders.new_table {
          results = colors,
        },
        sorter = conf.generic_sorter(opts),
        previewer = previewer,
        attach_mappings = function(prompt_bufnr)
          actions.select_default:replace(function()
            local selection = action_state.get_selected_entry()
            if selection == nil then
              utils.__warn_no_selection 'builtin.colorscheme'
              return
            end

            need_restore = false
            actions.close(prompt_bufnr)
            vim.cmd.colorscheme(selection.value)
          end)
          return true
        end,
        on_complete = {
          function()
            local selection = action_state.get_selected_entry()
            if selection == nil then
              utils.__warn_no_selection 'builtin.colorscheme'
              return
            end
            if opts.enable_preview then
              vim.cmd.colorscheme(selection.value)
            end
          end,
        },
      })

      if opts.enable_preview then
        -- rewrite picker.close_windows. restore color if needed
        local close_windows = picker.close_windows
        picker.close_windows = function(status)
          close_windows(status)
          if need_restore then
            vim.o.background = before_background
            vim.cmd.colorscheme(before_color)
          end
        end

        -- rewrite picker.set_selection so that color schemes can be previewed when the current
        -- selection is shifted using the keyboard or if an item is clicked with the mouse
        local set_selection = picker.set_selection
        picker.set_selection = function(self, row)
          set_selection(self, row)
          local selection = action_state.get_selected_entry()
          if selection == nil then
            utils.__warn_no_selection 'builtin.colorscheme'
            return
          end
          if opts.enable_preview then
            vim.cmd.colorscheme(selection.value)
          end
        end
      end

      picker:find()
    end, {
      nargs = 0,
    })

    vim.cmd.colorscheme(active_colorscheme)
  end,
}
