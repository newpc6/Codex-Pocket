<template>
  <div class="session-detail-page" :class="{ 'is-mobile': isMobile }">
    <div v-if="summary" class="session-hero">
      <div class="hero-top">
        <button type="button" class="back-chip" @click="$router.push('/')">
          <el-icon><ArrowLeft /></el-icon>
          <span>返回会话</span>
        </button>

        <div class="hero-actions">
          <el-button :icon="Refresh" :loading="app.loading" @click="refreshPage()" circle size="small" />
          <el-dropdown trigger="click" @command="onAction">
            <el-button size="small"><el-icon><More /></el-icon></el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="!summary.loaded && !summary.ended" command="resume">
                  <el-icon><Connection /></el-icon> 接管会话
                </el-dropdown-item>
                <el-dropdown-item v-if="summary.ended" command="resume">
                  <el-icon><Connection /></el-icon> 重新接管
                </el-dropdown-item>
                <el-dropdown-item v-if="summary.loaded && !summary.ended" command="detach">
                  <el-icon><SwitchButton /></el-icon> 取消接管
                </el-dropdown-item>
                <el-dropdown-item v-if="summary.loaded && !summary.ended" command="end">
                  <el-icon><SwitchButton /></el-icon> 结束会话
                </el-dropdown-item>
                <el-dropdown-item command="rename">重命名</el-dropdown-item>
                <el-dropdown-item v-if="summary.agentId === 'codex'" command="goal">设置目标</el-dropdown-item>
                <el-dropdown-item v-if="summary.agentId === 'codex' && detail?.goal" command="goal-clear">清空目标</el-dropdown-item>
                <el-dropdown-item v-if="summary.agentId === 'codex'" command="fork">分支会话</el-dropdown-item>
                <el-dropdown-item v-if="summary.agentId === 'codex'" command="compact">压缩上下文</el-dropdown-item>
                <el-dropdown-item v-if="summary.agentId === 'codex'" command="rollback">回滚最近一轮</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <div class="hero-main">
        <div class="hero-title-group">
          <div class="hero-name-row">
            <h1 class="hero-name">{{ displayName(summary) }}</h1>
            <el-tag :type="statusTagType(summary.status, summary.ended)" size="small" effect="light">
              {{ statusLabel(summary.status, summary.ended, summary.activeFlags?.length > 0) }}
            </el-tag>
            <div v-if="summary.lastTurnStatus === 'inProgress'" class="live-indicator">
              <span class="live-dot"></span>
              <span>执行中</span>
            </div>
          </div>

          <div class="hero-meta-row">
            <div class="hero-tags">
              <span class="hero-pill" :class="{ 'is-active': summary.loaded }">{{ summary.loaded ? '已接管' : '未接管' }}</span>
              <span v-if="summary.branch" class="hero-pill">{{ summary.branch }}</span>
              <span class="hero-pill">{{ lifecycleLabel(summary.lifecycleStage) }}</span>
            </div>
            <span class="hero-cwd">{{ summary.cwd }}</span>
          </div>

          <p v-if="summary.preview" class="hero-preview">
            {{ truncateText(summary.preview, 72) }}
          </p>
        </div>

        <div class="hero-status-card">
          <div class="hero-status-copy">
            <div class="hero-status-label">当前状态</div>
            <div class="hero-status-value">
              {{ summary.ended ? '会话已结束' : summary.loaded ? 'CodexPocket 正在托管' : '会话未接管' }}
            </div>
            <div class="hero-status-desc">
              {{ statusDescription(summary) }}
            </div>
          </div>

          <div class="hero-primary-actions">
            <el-button
              v-if="summary.agentId === 'codex'"
              size="small"
              :loading="reviewing"
              @click="openReviewDialog"
            >
              审查改动
            </el-button>
            <el-button
              v-if="!summary.loaded && !summary.ended"
              type="primary"
              size="small"
              :loading="resuming"
              @click="handleResume"
            >
              接管会话
            </el-button>
            <el-button
              v-else-if="summary.ended"
              type="primary"
              size="small"
              :loading="resuming"
              @click="handleResume"
            >
              重新接管
            </el-button>
            <el-button
              v-else
              size="small"
              :loading="detaching"
              @click="handleDetach"
            >
              取消接管
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <div class="content-area">
      <div v-if="detail?.goal" class="goal-card">
        <div class="goal-main">
          <div class="goal-label">当前目标</div>
          <div class="goal-objective">{{ detail.goal.objective }}</div>
          <div class="goal-meta">
            <span>{{ goalStatusLabel(detail.goal.status) }}</span>
            <span v-if="detail.goal.tokenBudget > 0">
              {{ detail.goal.tokensUsed }} / {{ detail.goal.tokenBudget }} tokens
            </span>
            <span v-if="detail.goal.timeUsedSeconds > 0">{{ formatGoalTime(detail.goal.timeUsedSeconds) }}</span>
          </div>
        </div>
        <div class="goal-actions">
          <el-button size="small" text @click="handleGoal">编辑</el-button>
          <el-button size="small" text type="danger" @click="handleClearGoal">清空</el-button>
        </div>
      </div>

      <div v-if="sessionApprovals.length > 0" class="approval-section">
        <div v-for="approval in sessionApprovals" :key="approval.id" class="approval-card">
          <div class="approval-info">
            <div class="approval-kind">{{ approval.kind }}</div>
            <div class="approval-reason">{{ approval.reason || approval.summary }}</div>
          </div>
          <div class="approval-actions">
            <el-button
              v-for="choice in approvalChoices(approval)"
              :key="choice.value"
              size="small"
              :type="choice.type"
              @click="handleApprovalChoice(approval, choice.value)"
            >
              {{ choice.label }}
            </el-button>
          </div>
        </div>
      </div>

      <div class="chat-shell">
        <div class="chat-toolbar">
          <div class="toolbar-left">
            <el-tag size="small" type="info" round>{{ detail?.totalTurns ?? orderedTurns.length }} 轮对话</el-tag>
            <span v-if="!followLiveOutput && latestTurn" class="follow-tip">已停留在历史位置</span>
          </div>
          <div class="toolbar-right">
            <el-button size="small" text :loading="changesLoading" @click="openChangesDrawer">
              文件变更
              <template v-if="changes?.summary.files">({{ changes.summary.files }})</template>
            </el-button>
            <el-button v-if="!followLiveOutput && latestTurn" size="small" text @click="jumpToLatest">回到最新</el-button>
          </div>
        </div>

        <div class="chat-area" ref="chatAreaRef" @scroll="onChatScroll">
          <div v-if="detail?.hasMoreHistory" class="history-load-row">
            <el-button text size="small" :loading="loadingHistory" @click="loadOlderTurns">加载更早对话</el-button>
          </div>

          <div v-if="isCompactingSession && !runningTurn" class="activity-row">
            <span class="activity-spinner"></span>
            <span>正在自动压缩上下文</span>
          </div>

          <div v-if="detail && detail.turns.length === 0" class="empty-hint">
            {{ summary?.ended ? '会话已结束，没有更多对话。' : '还没有对话，在下方发送指令开始。' }}
          </div>

          <template v-if="orderedTurns.length > 0">
            <section v-for="turn in orderedTurns" :key="turn.id" class="turn-stream">
              <div class="turn-anchor">
                <span class="turn-title">Turn #{{ turnNumber(turn.id) }}</span>
                <span class="turn-meta">{{ turnStatusText(turn) }}</span>
              </div>

              <div v-if="shouldShowTurnActivity(turn) && turnProcessSummaryItems(turn).length === 0" class="activity-row">
                <span class="activity-spinner"></span>
                <span>{{ liveActivityText(turn) }}</span>
              </div>

              <div
                v-for="entry in turnVisibleEntries(turn)"
                :key="entry.item.id || `${turn.id}-${entry.index}`"
                class="message-row"
                :class="messageSide(entry.item.type)"
              >
                <div class="message-bubble" :class="bubbleClass(entry.item.type)">
                  <div v-if="!isStructuredToolItem(entry.item)" class="message-topline">
                    <span class="message-label">{{ itemLabel(entry.item.type) }}</span>
                    <span v-if="entry.item.status" class="message-status">{{ entry.item.status }}</span>
                  </div>

                  <div
                    v-if="entry.item.title && entry.item.type !== 'userMessage' && entry.item.type !== 'agentMessage' && !isStructuredToolItem(entry.item)"
                    class="message-title"
                  >
                    {{ entry.item.title }}
                  </div>

                  <details
                    v-if="shouldRenderProcessInEntry(turn, entry)"
                    class="turn-process is-inline"
                    :open="turn.status === 'inProgress'"
                  >
                    <summary class="turn-process-summary">
                      <span class="turn-process-title">
                        <span v-if="turn.status === 'inProgress'" class="activity-spinner is-small"></span>
                        <span>{{ turnProcessedSummary(turn) }}</span>
                      </span>
                      <span v-if="turnProcessedDuration(turn)" class="turn-process-duration">{{ turnProcessedDuration(turn) }}</span>
                    </summary>

                    <div class="turn-process-items">
                      <div v-if="turnProcessFileEditSummary(turn).files.length > 0" class="process-entry-card is-live-file-edit">
                        <div class="process-entry-head">
                          <span>{{ turnProcessFileEditSummary(turn).label }}</span>
                          <span class="file-edit-stats">
                            <span class="diff-add">+{{ turnProcessFileEditSummary(turn).additions }}</span>
                            <span class="diff-del">-{{ turnProcessFileEditSummary(turn).deletions }}</span>
                          </span>
                        </div>
                        <div class="file-edit-list">
                          <div
                            v-for="file in turnProcessFileEditSummary(turn).files.slice(0, 3)"
                            :key="`${turn.id}-process-file-${file.path}`"
                            class="file-edit-row"
                          >
                            <span class="file-edit-path">{{ file.path }}</span>
                            <span class="file-edit-stats">
                              <span class="diff-add">+{{ file.additions }}</span>
                              <span class="diff-del">-{{ file.deletions }}</span>
                            </span>
                          </div>
                        </div>
                      </div>
                      <template
                        v-for="block in turnTimelineBlocks(turn)"
                        :key="`${turn.id}-timeline-${block.startIndex}-${block.kind}`"
                      >
                        <div v-if="block.kind === 'commands'" class="process-command-group">
                          <details>
                            <summary class="process-command-summary">
                              <span>已运行 {{ block.entries.length }} 条命令</span>
                              <span v-if="blockDuration(block)">{{ blockDuration(block) }}</span>
                            </summary>
                            <div class="process-command-list">
                              <div
                                v-for="processEntry in block.entries"
                                :key="processEntry.item.id || `${turn.id}-cmd-${processEntry.index}`"
                                class="process-entry-card"
                              >
                                <div class="process-entry-head">
                                  <span>{{ toolDisplayName(processEntry.item) }}</span>
                                  <span v-if="toolCommandTag(processEntry.item)" class="tool-command-tag" :title="toolCommandTag(processEntry.item)">
                                    {{ toolCommandTag(processEntry.item) }}
                                  </span>
                                </div>
                                <details v-if="hasStructuredToolDetails(processEntry.item)" class="tool-details">
                                  <summary>输出</summary>
                                  <div v-if="processEntry.item.body" class="message-body is-code">
                                    <pre>{{ processEntry.item.body }}</pre>
                                  </div>
                                  <div v-if="processEntry.item.auxiliary" class="message-aux tool-output">
                                    <pre>{{ processEntry.item.auxiliary }}</pre>
                                  </div>
                                </details>
                              </div>
                            </div>
                          </details>
                        </div>

                        <div
                          v-else
                          v-for="processEntry in block.entries"
                          :key="processEntry.item.id || `${turn.id}-timeline-${processEntry.index}`"
                          class="process-entry-card"
                          :class="`is-${processEntry.item.type}`"
                        >
                          <div
                            v-if="processEntry.item.type !== 'agentMessage' && processEntry.item.type !== 'userMessage'"
                            class="process-entry-head"
                          >
                            <span>{{ processEntryTitle(processEntry.item) }}</span>
                            <span v-if="processEntry.item.status">{{ processEntry.item.status }}</span>
                          </div>
                          <template v-if="processEntry.item.type === 'agentMessage' || processEntry.item.type === 'userMessage'">
                            <div v-if="itemImages(processEntry.item).length" class="image-strip">
                              <el-image
                                v-for="image in itemImages(processEntry.item)"
                                :key="image.url"
                                class="message-thumb"
                                :src="image.url"
                                :preview-src-list="itemPreviewUrls(processEntry.item)"
                                :initial-index="image.index"
                                fit="cover"
                                preview-teleported
                                @load="handleMessageAssetLoad"
                                @error="handleMessageAssetError"
                              />
                            </div>
                            <div v-if="itemText(processEntry.item)" class="markdown-body">
                              <VueMarkdown :source="renderMarkdown(itemText(processEntry.item))" :options="markdownOptions" />
                            </div>
                          </template>
                          <template v-else-if="isStructuredToolItem(processEntry.item)">
                            <div class="tool-headline">
                              <span class="tool-type">{{ toolDisplayName(processEntry.item) }}</span>
                              <span v-if="toolCommandTag(processEntry.item)" class="tool-command-tag" :title="toolCommandTag(processEntry.item)">
                                {{ toolCommandTag(processEntry.item) }}
                              </span>
                            </div>
                            <details v-if="hasStructuredToolDetails(processEntry.item)" class="tool-details">
                              <summary>查看原始内容</summary>
                              <div v-if="processEntry.item.body" class="message-body is-code">
                                <pre>{{ processEntry.item.body }}</pre>
                              </div>
                              <div v-if="processEntry.item.auxiliary" class="message-aux tool-output">
                                <pre>{{ processEntry.item.auxiliary }}</pre>
                              </div>
                            </details>
                          </template>
                          <template v-else>
                            <div v-if="processEntry.item.body" class="message-body" :class="{ 'is-code': isCodeType(processEntry.item.type) }">
                              <pre v-if="isCodeType(processEntry.item.type)">{{ processEntry.item.body }}</pre>
                              <div v-else class="markdown-body">
                                <VueMarkdown :source="renderMarkdown(itemText(processEntry.item) || processEntry.item.body)" :options="markdownOptions" />
                              </div>
                            </div>
                            <details v-if="processEntry.item.auxiliary" class="message-aux">
                              <summary>详细输出</summary>
                              <pre>{{ processEntry.item.auxiliary }}</pre>
                            </details>
                          </template>
                        </div>

                        <div v-if="block.kind === 'diff'" class="process-entry-card is-fileChange">
                          <div class="process-entry-head">
                            <span>文件变更</span>
                            <span>
                              <span class="diff-add">+{{ diffSummary(turn.diff).additions }}</span>
                              <span class="diff-del"> -{{ diffSummary(turn.diff).deletions }}</span>
                            </span>
                          </div>
                          <details class="tool-details">
                            <summary>查看 diff</summary>
                            <div class="file-change-list">
                              <div v-for="file in diffSummary(turn.diff).files" :key="file.path" class="file-change-row">
                                <span class="file-change-path">{{ file.path }}</span>
                                <span class="file-change-stats">
                                  <span class="diff-add">+{{ file.additions }}</span>
                                  <span class="diff-del">-{{ file.deletions }}</span>
                                </span>
                              </div>
                            </div>
                            <div class="diff-viewer inline-diff-viewer">
                              <div
                                v-for="(line, index) in diffLines(turn.diff)"
                                :key="`turn-diff-${turn.id}-${index}`"
                                class="diff-line"
                                :class="diffLineClass(line)"
                              >
                                {{ line }}
                              </div>
                            </div>
                          </details>
                        </div>
                      </template>

                      <div v-if="shouldShowInlineLiveStatus(turn)" class="process-live-row">
                        <span class="activity-spinner is-small"></span>
                        <span>{{ liveActivityText(turn) }}</span>
                      </div>
                    </div>
                  </details>

                  <div
                    v-if="shouldRenderProcessInEntry(turn, entry) && (entry.item.body || entry.item.auxiliary)"
                    class="process-summary-divider"
                  ></div>

                  <template v-if="isStructuredToolItem(entry.item)">
                    <div class="tool-card">
                      <div class="tool-summary">
                        <div class="tool-main">
                          <div class="tool-name">工具</div>
                          <div class="tool-headline">
                            <span class="tool-type">{{ toolDisplayName(entry.item) }}</span>
                            <span
                              v-if="toolCommandTag(entry.item)"
                              class="tool-command-tag"
                              :title="toolCommandTag(entry.item)"
                            >
                              {{ toolCommandTag(entry.item) }}
                            </span>
                          </div>
                        </div>
                      </div>

                      <details v-if="hasStructuredToolDetails(entry.item)" class="tool-details">
                        <summary>查看原始内容</summary>
                        <div v-if="entry.item.body" class="message-body is-code">
                          <pre>{{ entry.item.body }}</pre>
                        </div>
                        <div v-if="entry.item.auxiliary" class="message-aux tool-output">
                          <div class="tool-output-title">输出</div>
                          <pre>{{ entry.item.auxiliary }}</pre>
                        </div>
                      </details>
                    </div>
                  </template>

                  <template v-else>
                    <div v-if="entry.item.body" class="message-body" :class="{ 'is-code': isCodeType(entry.item.type) }">
                      <pre v-if="isCodeType(entry.item.type)">{{ entry.item.body }}</pre>
                      <template v-else>
                        <div v-if="itemImages(entry.item).length" class="image-strip">
                          <el-image
                            v-for="image in itemImages(entry.item)"
                            :key="image.url"
                            class="message-thumb"
                            :src="image.url"
                            :preview-src-list="itemPreviewUrls(entry.item)"
                            :initial-index="image.index"
                            fit="cover"
                            preview-teleported
                            @load="handleMessageAssetLoad"
                            @error="handleMessageAssetError"
                          />
                        </div>
                        <div v-if="itemText(entry.item)" class="markdown-body">
                          <VueMarkdown :source="renderMarkdown(itemText(entry.item))" :options="markdownOptions" />
                          <span v-if="isStreamingItem(turn, entry.item, entry.index)" class="typing-cursor">|</span>
                        </div>
                      </template>
                    </div>

                    <details v-if="entry.item.auxiliary" class="message-aux">
                      <summary>详细输出</summary>
                      <pre>{{ entry.item.auxiliary }}</pre>
                    </details>
                  </template>
                </div>
              </div>

              <div v-if="turn.error" class="message-row side-left">
                <div class="message-bubble bubble-error">
                  <div class="message-topline">
                    <span class="message-label">错误</span>
                  </div>
                  <div class="message-body">{{ turn.error }}</div>
                </div>
              </div>

              <div v-if="turnDisplayChangedFiles(turn).length > 0" class="turn-change-card">
                <div class="turn-change-head">
                  <span>{{ turnChangeSummaryText(turn) }}</span>
                  <div class="turn-change-actions">
                    <button type="button" @click="reviewTurnChanges(turn)">审查</button>
                    <button
                      v-if="turn.status !== 'inProgress'"
                      type="button"
                      :disabled="revertingFiles"
                      @click="revertTurnChanges(turn)"
                    >
                      撤销
                    </button>
                  </div>
                </div>
                <div class="turn-change-list">
                  <button
                    v-for="file in visibleTurnChangedFiles(turn)"
                    :key="`${turn.id}-${file.path}`"
                    type="button"
                    class="turn-change-row"
                    @click="openTurnChangedFile(turn, file.path)"
                  >
                    <span class="changed-file-path">{{ file.path }}</span>
                    <span class="changed-file-stats">
                      <span class="diff-add">+{{ file.additions }}</span>
                      <span class="diff-del">-{{ file.deletions }}</span>
                    </span>
                  </button>
                  <button
                    v-if="hiddenTurnChangeCount(turn) > 0"
                    type="button"
                    class="turn-change-more"
                    @click="toggleTurnChangeExpanded(turn.id)"
                  >
                    <span>{{ expandedTurnChangeIds.has(turn.id) ? '收起文件' : `再显示 ${hiddenTurnChangeCount(turn)} 个文件` }}</span>
                    <el-icon><ArrowRight /></el-icon>
                  </button>
                </div>
              </div>
            </section>
          </template>

          <div v-else-if="!app.loading && !detail" class="empty-hint">
            <el-icon class="is-loading" :size="20"><Loading /></el-icon>
            <span>正在加载…</span>
          </div>
        </div>
        <transition name="new-message-pill">
          <button
            v-if="showNewMessageHint"
            type="button"
            class="new-message-pill"
            @click="jumpToLatest"
          >
            有新消息，回到最新
          </button>
        </transition>
      </div>
    </div>

    <el-drawer
      v-model="changesDrawerOpen"
      :direction="isMobile ? 'btt' : 'rtl'"
      :size="isMobile ? '82%' : '560px'"
      title="文件变更"
      class="changes-drawer"
    >
      <div class="changes-panel">
        <div class="change-scope-bar">
          <div v-if="changeScope === 'turn'" class="turn-scope-chip">本轮改动</div>
          <el-segmented v-else v-model="changeScope" :options="changeScopeOptions" @change="reloadChanges" />
          <el-input
            v-if="changeScope === 'commit'"
            v-model="changeRef"
            placeholder="commit hash"
            clearable
            @keyup.enter="reloadChanges"
          />
          <el-input
            v-if="changeScope === 'base'"
            v-model="changeBase"
            placeholder="base branch，例如 main"
            clearable
            @keyup.enter="reloadChanges"
          />
          <el-button :icon="Refresh" :loading="changesLoading" circle @click="reloadChanges" />
        </div>

        <el-alert
          v-if="changesError"
          :title="changesError"
          type="warning"
          show-icon
          :closable="false"
        />

        <div v-if="changes" class="changes-summary-row">
          <span>已编辑 {{ changes.summary.files }} 个文件</span>
          <span class="diff-add">+{{ changes.summary.additions }}</span>
          <span class="diff-del">-{{ changes.summary.deletions }}</span>
          <span v-if="changes.summary.untracked > 0">{{ changes.summary.untracked }} 未跟踪</span>
        </div>

        <div v-if="changes?.files.length" class="changed-file-list">
          <button
            v-for="file in changes.files"
            :key="file.path"
            type="button"
            class="changed-file-row"
            :class="{ 'is-selected': selectedChangeFile === file.path }"
            @click="selectChangedFile(file.path)"
          >
            <span class="changed-file-status">{{ file.status }}</span>
            <span class="changed-file-path">{{ file.path }}</span>
            <span class="changed-file-stats">
              <span class="diff-add">+{{ file.additions }}</span>
              <span class="diff-del">-{{ file.deletions }}</span>
            </span>
          </button>
        </div>

        <el-empty v-else-if="!changesLoading && !changesError" description="当前范围没有文件变更" />

        <div v-if="selectedFileDetail" class="file-detail-panel">
          <div class="file-detail-head">
            <div class="file-detail-title">{{ selectedFileDetail.path }}</div>
            <el-radio-group v-model="fileViewMode" size="small">
              <el-radio-button label="diff">Diff</el-radio-button>
              <el-radio-button label="content">当前文件</el-radio-button>
            </el-radio-group>
          </div>
          <div v-if="fileViewMode === 'diff'" class="diff-viewer file-detail-code">
            <template v-if="diffLines(selectedFileDetail.diff).length">
              <div
                v-for="(line, index) in diffLines(selectedFileDetail.diff)"
                :key="`${selectedFileDetail.path}-${index}`"
                class="diff-line"
                :class="diffLineClass(line)"
              >
                {{ line || ' ' }}
              </div>
            </template>
            <div v-else class="diff-empty">没有可显示的 diff</div>
          </div>
          <pre v-else-if="selectedFileDetail.readable" class="file-content-block file-detail-code">{{ selectedFileDetail.content }}</pre>
          <div v-else class="file-readable-error">{{ selectedFileDetail.error || '无法预览这个文件' }}</div>
          <div v-if="selectedFileDetail.truncated" class="file-truncated-note">文件较大，已截断预览。</div>
        </div>
      </div>
    </el-drawer>

    <el-dialog
      v-model="reviewDialogOpen"
      title="审查改动"
      :width="isMobile ? '92%' : '760px'"
      class="review-dialog"
      :close-on-click-modal="false"
    >
      <el-form label-width="82px">
        <el-form-item v-if="reviewScope === 'turn'" label="范围">
          <div class="turn-scope-chip">本轮改动</div>
        </el-form-item>
        <el-form-item v-else label="范围">
          <el-segmented v-model="reviewScope" :options="changeScopeOptions" @change="reloadReviewPreview" />
        </el-form-item>
        <el-form-item v-if="reviewScope === 'commit'" label="Commit">
          <el-input v-model="reviewRef" placeholder="commit hash" @keyup.enter="reloadReviewPreview" />
        </el-form-item>
        <el-form-item v-if="reviewScope === 'base'" label="Base">
          <el-input v-model="reviewBase" placeholder="main / develop / origin/main" @keyup.enter="reloadReviewPreview" />
        </el-form-item>
      </el-form>

      <div class="review-preview">
        <div class="review-preview-head">
          <div v-if="reviewPreview" class="changes-summary-row is-review">
            <span>已编辑 {{ reviewPreview.summary.files }} 个文件</span>
            <span class="diff-add">+{{ reviewPreview.summary.additions }}</span>
            <span class="diff-del">-{{ reviewPreview.summary.deletions }}</span>
          </div>
          <el-button size="small" :icon="Refresh" :loading="reviewPreviewLoading" @click="reloadReviewPreview">刷新</el-button>
        </div>
        <el-alert
          v-if="reviewPreviewError"
          :title="reviewPreviewError"
          type="warning"
          show-icon
          :closable="false"
        />
        <div v-else-if="reviewPreviewLoading && !reviewPreview" class="review-loading">正在加载改动…</div>
        <el-empty v-else-if="reviewPreview && reviewPreview.files.length === 0" description="当前范围没有代码改动" />
        <template v-else-if="reviewPreview">
          <div class="review-file-strip">
            <button
              v-for="file in reviewPreview.files"
              :key="file.path"
              type="button"
              class="review-file-chip"
              :class="{ 'is-selected': selectedReviewFile === file.path }"
              @click="selectReviewFile(file.path)"
            >
              <span>{{ file.path }}</span>
              <span>
                <span class="diff-add">+{{ file.additions }}</span>
                <span class="diff-del">-{{ file.deletions }}</span>
              </span>
            </button>
          </div>
          <div class="diff-viewer review-diff-viewer">
            <template v-if="reviewDiffLines.length">
              <div
                v-for="(line, index) in reviewDiffLines"
                :key="`review-${selectedReviewFile}-${index}`"
                class="diff-line"
                :class="diffLineClass(line)"
              >
                {{ line || ' ' }}
              </div>
            </template>
            <div v-else class="diff-empty">请选择文件查看 diff</div>
          </div>
        </template>
      </div>
      <template #footer>
        <el-button @click="reviewDialogOpen = false">取消</el-button>
        <el-button type="primary" :loading="reviewing" @click="handleStartReview">开始审查</el-button>
      </template>
    </el-dialog>

    <div v-if="summary && !summary.ended" class="input-area">
      <div v-if="!summary.loaded" class="input-status-hint">
        未接管会话，发送时会自动接管。
      </div>
      <div v-if="pendingImages.length > 0" class="pending-image-row">
        <div v-for="image in pendingImages" :key="image.id" class="pending-image-chip">
          <span>{{ image.name }}</span>
          <button type="button" @click="removePendingImage(image.id)">移除</button>
        </div>
      </div>
      <div class="input-row">
        <el-dropdown trigger="click" @command="onInputAction">
          <el-button class="input-plus-btn" :icon="Plus" circle :disabled="submitting" />
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="image">上传图片</el-dropdown-item>
              <el-dropdown-item command="changes">文件变更</el-dropdown-item>
              <el-dropdown-item command="review">审查改动</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <input
          ref="imageInputRef"
          class="hidden-file-input"
          type="file"
          accept="image/*"
          multiple
          @change="handleImageFiles"
        />
        <el-input
          v-model="promptText"
          type="textarea"
          :autosize="{ minRows: 1, maxRows: 4 }"
          placeholder="输入指令…"
          :disabled="submitting"
          @keydown.enter.exact.prevent="handleSubmit"
        />
        <el-button type="primary" :loading="submitting || uploadingImage" @click="handleSubmit"
          :disabled="!promptText.trim() && pendingImages.length === 0" class="send-btn">
          {{ sendButtonLabel }}
        </el-button>
        <el-button v-if="runningTurn" type="warning" size="small" @click="handleInterrupt">
          中断
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore, type ApprovalRequest, type ChangedFileDetail, type SessionChanges, type SessionSummary, type Turn, type TurnItem } from '../stores/app'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Refresh, More, ArrowRight, Connection, SwitchButton, Loading, Plus } from '@element-plus/icons-vue'
import VueMarkdown from 'vue-markdown-render'
import {
  formatTimestamp, statusTagType, statusLabel, lifecycleLabel,
  lifecycleTagType, truncateText, sessionDisplayName,
} from '../utils/helpers'
import api from '../utils/api'

const route = useRoute()
const router = useRouter()
const app = useAppStore()
const sessionId = route.params.id as string
const promptText = ref('')
const submitting = ref(false)
const uploadingImage = ref(false)
const resuming = ref(false)
const detaching = ref(false)
const reviewing = ref(false)
const revertingFiles = ref(false)
const chatAreaRef = ref<HTMLElement | null>(null)
const imageInputRef = ref<HTMLInputElement | null>(null)
const followLiveOutput = ref(true)
const loadingHistory = ref(false)
const pendingNewMessages = ref(0)
const changesDrawerOpen = ref(false)
const changesLoading = ref(false)
const changesError = ref('')
const changes = ref<SessionChanges | null>(null)
const liveChanges = ref<SessionChanges | null>(null)
const selectedFileDetail = ref<ChangedFileDetail | null>(null)
const selectedChangeFile = ref('')
const changeScope = ref('workspace')
const changeRef = ref('')
const changeBase = ref('main')
const activeTurnChangeId = ref('')
const fileViewMode = ref<'diff' | 'content'>('diff')
const reviewDialogOpen = ref(false)
const reviewScope = ref('workspace')
const reviewRef = ref('')
const reviewBase = ref('main')
const activeReviewTurnId = ref('')
const reviewPreview = ref<SessionChanges | null>(null)
const reviewPreviewLoading = ref(false)
const reviewPreviewError = ref('')
const selectedReviewFile = ref('')
const selectedReviewFileDetail = ref<ChangedFileDetail | null>(null)
const expandedTurnChangeIds = ref(new Set<string>())
const pendingImages = ref<Array<{ id: string; name: string; size: number }>>([])
const tabInstanceId = `${Date.now()}-${Math.random().toString(36).slice(2)}`
let liveSyncTimer: ReturnType<typeof setInterval> | null = null
let liveChangesTimer: ReturnType<typeof setInterval> | null = null
let elapsedTimer: ReturnType<typeof setInterval> | null = null
let liveSyncBusy = false
let liveChangesBusy = false
let initialScrollDone = false

const liveLeaseKey = `cf_live_session_lease:${sessionId}`
const liveSnapshotKey = `cf_live_session_snapshot:${sessionId}`
const liveLeaseMs = 2600
const liveSyncIntervalMs = 900

const markdownOptions = {
  html: false,
  breaks: true,
  linkify: true,
  typographer: true,
}

const localAssetBase = '/api/v1/assets/local-image'
type TurnItemEntry = { item: TurnItem; index: number }
type TurnTimelineBlock = {
  kind: 'entries' | 'commands' | 'diff'
  entries: TurnItemEntry[]
  startIndex: number
}
type DiffFileSummary = { path: string; additions: number; deletions: number }
type DiffSummary = { files: DiffFileSummary[]; additions: number; deletions: number }
type MessageImage = { url: string; alt: string; index: number }
const turnChangePreviewLimit = 2
const diffSummaryCache = new Map<string, DiffSummary>()
const changeScopeOptions = [
  { label: '工作区', value: 'workspace' },
  { label: 'Commit', value: 'commit' },
  { label: 'Base', value: 'base' },
]

const isMobile = ref(window.innerWidth <= 768)
function onResize() { isMobile.value = window.innerWidth <= 768 }
window.addEventListener('resize', onResize)

const detail = computed(() => app.sessionDetails[sessionId])
const summary = computed<SessionSummary | undefined>(() => {
  if (detail.value) return detail.value.summary
  return app.dashboard.sessions.find((s) => s.id === sessionId)
})

const sessionApprovals = computed(() => app.filteredApprovals.filter((a) => a.threadId === sessionId))
const orderedTurns = computed(() => detail.value?.turns || [])
const latestTurn = computed(() => orderedTurns.value[orderedTurns.value.length - 1])
const elapsedNow = ref(Date.now())
const showNewMessageHint = computed(() => pendingNewMessages.value > 0 && !followLiveOutput.value)
const runningTurn = computed(() => {
  for (let i = orderedTurns.value.length - 1; i >= 0; i -= 1) {
    const turn = orderedTurns.value[i]
    if (turn.status === 'inProgress') return turn
  }
  return undefined
})
const reviewDiffLines = computed(() => diffLines(selectedReviewFileDetail.value?.diff || ''))
const isStreamingReply = computed(() => {
  const turn = runningTurn.value
  if (!turn) return false
  return turn.items?.some((item: TurnItem) => item.type === 'agentMessage' && item.body)
})
const isCompactingSession = computed(() => app.compactingSessionIds.has(sessionId))
const sendButtonLabel = computed(() => {
  if (runningTurn.value) return 'Steer'
  if (!summary.value?.loaded) return '接管并发送'
  return '发送'
})

function displayName(s: SessionSummary) { return sessionDisplayName(s) }

function statusDescription(s: SessionSummary) {
  if (s.ended) return '当前会话已经结束，但历史内容仍然保留，可以重新接管继续工作。'
  if (s.loaded) return '当前会话由 CodexPocket 持续同步和控制，你可以在这里继续发送指令或中断执行。'
  return '当前会话还没有由 CodexPocket 托管，接管后可以继续执行并实时查看消息。'
}

function itemLabel(type: string): string {
  switch (type) {
    case 'userMessage': return '用户'
    case 'agentMessage': return 'Codex'
    case 'commandExecution': return '命令执行'
    case 'fileChange': return '文件变更'
    case 'reasoning': return '思考'
    case 'plan': return '计划'
    case 'mcpToolCall': return 'MCP 工具'
    case 'dynamicToolCall': return '工具'
    case 'collabAgentToolCall': return '协作'
    default: return type
  }
}

function messageSide(type: string) {
  return type === 'userMessage' ? 'side-right' : 'side-left'
}

function bubbleClass(type: string) {
  switch (type) {
    case 'userMessage': return 'bubble-user'
    case 'agentMessage': return 'bubble-agent'
    case 'commandExecution':
    case 'dynamicToolCall':
    case 'mcpToolCall':
    case 'collabAgentToolCall':
    case 'fileChange':
      return 'bubble-tool'
    case 'reasoning':
    case 'plan':
      return 'bubble-meta'
    default:
      return 'bubble-other'
  }
}

function isCodeType(type: string): boolean {
  return ['commandExecution', 'fileChange', 'mcpToolCall', 'dynamicToolCall'].includes(type)
}

function isStructuredToolItem(item: TurnItem): boolean {
  return item.type === 'commandExecution' || item.type === 'dynamicToolCall'
}

function toolDisplayName(item: TurnItem): string {
  if (item.type === 'commandExecution') return (item.title || 'shell_command').trim() || 'shell_command'
  const raw = item.title || item.type
  return raw.trim() || item.type
}

function toolCommandTag(item: TurnItem): string {
  const metadataCommand = (item.metadata?.command || '').trim()
  if (metadataCommand) return metadataCommand
  if (item.type === 'commandExecution') return (item.body || '').trim()
  if (item.type === 'dynamicToolCall' && toolDisplayName(item) === 'shell_command') {
    return extractCommandFromToolBody(item.body)
  }
  return ''
}

function extractCommandFromToolBody(body: string): string {
  const raw = (body || '').trim()
  if (!raw) return ''
  try {
    const decoded = JSON.parse(raw)
    return typeof decoded?.command === 'string' ? decoded.command.trim() : ''
  } catch {
    return ''
  }
}

function hasStructuredToolDetails(item: TurnItem): boolean {
  return Boolean((item.body && item.body.trim()) || (item.auxiliary && item.auxiliary.trim()))
}

function turnItemEntries(turn: Turn): TurnItemEntry[] {
  return (turn.items || []).map((item, index) => ({ item, index }))
}

function isPrimaryItem(item: TurnItem): boolean {
  return item.type === 'userMessage' || item.type === 'agentMessage'
}

function isInjectedUserMessage(item: TurnItem): boolean {
  if (item.type !== 'userMessage') return false
  if (item.metadata?.localInput === 'true') return false
  const text = itemText(item) || item.body || ''
  return text.includes('<environment_context>')
    || text.includes('AGENTS.md instructions for')
    || text.includes('<INSTRUCTIONS>')
}

function turnPrimaryEntries(turn: Turn): TurnItemEntry[] {
  return turnItemEntries(turn).filter((entry) => isPrimaryItem(entry.item))
}

function turnProcessEntries(turn: Turn): TurnItemEntry[] {
  return turnItemEntries(turn).filter((entry) => !isPrimaryItem(entry.item))
}

function finalAgentEntry(turn: Turn): TurnItemEntry | undefined {
  const entries = turnItemEntries(turn)
  for (let i = entries.length - 1; i >= 0; i -= 1) {
    const entry = entries[i]
    if (entry.item.type === 'agentMessage' && itemText(entry.item)) {
      return entry
    }
  }
  return undefined
}

function processHostEntry(turn: Turn): TurnItemEntry | undefined {
  const finalEntry = finalAgentEntry(turn)
  if (finalEntry) return finalEntry
  if (!hasTurnProcessContent(turn)) return undefined
  return {
    index: Number.MAX_SAFE_INTEGER,
    item: {
      id: `${turn.id}-process-host`,
      type: 'agentMessage',
      title: '',
      body: '',
      status: '',
      auxiliary: '',
    },
  }
}

function turnVisibleEntries(turn: Turn): TurnItemEntry[] {
  const entries = turnItemEntries(turn)
  const userEntries = entries.filter((entry) => entry.item.type === 'userMessage' && !isInjectedUserMessage(entry.item))
  const hostEntry = processHostEntry(turn)
  if (hostEntry) return [...userEntries, hostEntry]
  return userEntries.length > 0
    ? userEntries
    : entries.filter((entry) => entry.item.type === 'agentMessage').slice(-1)
}

function shouldRenderProcessInEntry(turn: Turn, entry: TurnItemEntry): boolean {
  if (!hasTurnProcessContent(turn)) return false
  const hostEntry = processHostEntry(turn)
  return Boolean(hostEntry && entry.index === hostEntry.index && entry.item.id === hostEntry.item.id)
}

function hasTurnProcessContent(turn: Turn): boolean {
  return turnProcessSummaryItems(turn).length > 0
    || turnProcessFileEditSummary(turn).files.length > 0
    || hasInlineLiveStatus(turn)
}

function turnProcessSummaryItems(turn: Turn): TurnItemEntry[] {
  const finalEntry = finalAgentEntry(turn)
  const finalIndex = finalEntry?.index ?? -1
  return turnItemEntries(turn).filter((entry) => {
    if (entry.index === finalIndex) return false
    if (entry.item.type === 'userMessage') return false
    if (entry.item.type === 'fileChange') return false
    if (entry.item.type === 'agentMessage' && !itemText(entry.item)) return false
    return true
  })
}

function turnTimelineBlocks(turn: Turn): TurnTimelineBlock[] {
  const blocks: TurnTimelineBlock[] = []
  const summaryEntries = turnProcessSummaryItems(turn)
  let commandBlock: TurnTimelineBlock | null = null

  const flushCommandBlock = () => {
    if (commandBlock) {
      blocks.push(commandBlock)
      commandBlock = null
    }
  }

  for (const entry of summaryEntries) {
    if (isCommandLikeItem(entry.item)) {
      if (!commandBlock) {
        commandBlock = { kind: 'commands', entries: [], startIndex: entry.index }
      }
      commandBlock.entries.push(entry)
      continue
    }
    flushCommandBlock()
    blocks.push({ kind: 'entries', entries: [entry], startIndex: entry.index })
  }
  flushCommandBlock()

  if (turn.diff?.trim()) {
    blocks.push({ kind: 'diff', entries: [], startIndex: Number.MAX_SAFE_INTEGER })
  }
  return blocks
}

function isCommandLikeItem(item: TurnItem): boolean {
  return item.type === 'commandExecution'
    || item.type === 'mcpToolCall'
    || item.type === 'dynamicToolCall'
    || item.type === 'collabAgentToolCall'
}

function turnProcessSummary(turn: Turn): string {
  const entries = turnProcessEntries(turn)
  const commandCount = entries.filter((entry) => isCommandLikeItem(entry.item)).length
  const fileChangeCount = entries.filter((entry) => entry.item.type === 'fileChange').length + diffSummary(turn.diff).files.length
  const otherCount = Math.max(entries.length - commandCount - fileChangeCount, 0)
  const parts: string[] = []
  if (turn.status === 'inProgress') parts.push(liveActivityText(turn))
  if (commandCount > 0) parts.push(`已运行 ${commandCount} 条命令`)
  if (fileChangeCount > 0) parts.push(`修改 ${fileChangeCount} 个文件`)
  if (otherCount > 0) parts.push(`${otherCount} 条过程`)
  return parts.join(' · ') || `${entries.length} 条过程`
}

function liveChangedFilesForTurn(turn: Turn): DiffFileSummary[] {
  if (turn.status !== 'inProgress' || runningTurn.value?.id !== turn.id) return []
  return filterDisplayChangedFiles((liveChanges.value?.files || []).map((file) => ({
    path: file.path,
    additions: file.additions,
    deletions: file.deletions,
  })))
}

function turnDisplayChangedFiles(turn: Turn): DiffFileSummary[] {
  const base = turnChangedFiles(turn)
  const live = liveChangedFilesForTurn(turn)
  if (live.length === 0) return base
  const byPath = new Map<string, DiffFileSummary>()
  for (const file of [...base, ...live]) {
    const existing = byPath.get(file.path)
    if (existing) {
      existing.additions = Math.max(existing.additions, file.additions)
      existing.deletions = Math.max(existing.deletions, file.deletions)
    } else {
      byPath.set(file.path, { ...file })
    }
  }
  return Array.from(byPath.values())
}

function turnChangeSummaryText(turn: Turn): string {
  const files = turnDisplayChangedFiles(turn)
  const additions = files.reduce((sum, file) => sum + file.additions, 0)
  const deletions = files.reduce((sum, file) => sum + file.deletions, 0)
  const prefix = turn.status === 'inProgress' ? '当前任务' : ''
  return `${prefix}${files.length} 个文件已更改 +${additions} -${deletions}`
}

function turnProcessFileEditSummary(turn: Turn) {
  const files = turnDisplayChangedFiles(turn)
  const additions = files.reduce((sum, file) => sum + file.additions, 0)
  const deletions = files.reduce((sum, file) => sum + file.deletions, 0)
  return {
    files,
    additions,
    deletions,
    label: turn.status === 'inProgress'
      ? (files.length === 1 ? '正在编辑文件' : `正在编辑 ${files.length} 个文件`)
      : `已编辑 ${files.length} 个文件`,
  }
}

function turnProcessedSummary(turn: Turn): string {
  return '已处理'
}

function turnProcessedDuration(turn: Turn): string {
  if (turn.durationMs > 0) return formatDurationMs(turn.durationMs)
  if (turn.status === 'inProgress' && turn.startedAt > 0) {
    const startedAtMs = normalizeTimestampMs(turn.startedAt)
    return formatDurationMs(Math.max(elapsedNow.value - startedAtMs, 0))
  }
  return ''
}

function normalizeTimestampMs(value: number): number {
  if (!value || value <= 0) return 0
  return value < 1_000_000_000_000 ? value * 1000 : value
}

function processEntryTitle(item: TurnItem): string {
  if (item.type === 'fileChange') {
    const fileCount = fileChangeCountFromItem(item)
    return fileCount > 0 ? `已编辑 ${fileCount} 个文件` : '已编辑文件'
  }
  return itemLabel(item.type)
}

function fileChangeCountFromItem(item: TurnItem): number {
  const text = `${item.title || ''}\n${item.body || ''}\n${item.auxiliary || ''}`
  const match = text.match(/(\d+)\s*(?:个)?文件/)
  if (match) return Number(match[1]) || 0
  return 0
}

function blockDuration(_block: TurnTimelineBlock): string {
  return ''
}

function liveActivityText(turn: Turn): string {
  if (isCompactingSession.value) return '正在自动压缩上下文'
  if (isEditingFiles(turn)) return '正在编辑文件'
  if (turn.items.some((item) => isCommandLikeItem(item))) return '正在运行命令'
  if (turn.items.some((item) => item.type === 'agentMessage' && item.body)) return 'Codex 正在回复'
  if (turn.items.some((item) => item.type === 'reasoning')) return '正在思考'
  return '正在思考'
}

function shortFileName(path: string): string {
  const normalized = normalizeChangedPath(path)
  const parts = normalized.split('/').filter(Boolean)
  return parts[parts.length - 1] || normalized
}

function turnStatusText(turn: Turn): string {
  if (turn.status === 'inProgress') return liveActivityText(turn)
  return formatTimestamp(turn.startedAt)
}

function shouldShowTurnActivity(turn: Turn): boolean {
  if (turn.status !== 'inProgress') return false
  if (shouldShowInlineLiveStatus(turn)) return false
  if (isCompactingSession.value || isEditingFiles(turn)) return true
  if (turn.items.some((item) => isCommandLikeItem(item))) return true
  return !turn.items.some((item) => item.type === 'agentMessage' && item.body)
}

function shouldShowInlineLiveStatus(turn: Turn): boolean {
  if (turn.status !== 'inProgress') return false
  if (turnProcessFileEditSummary(turn).files.length > 0) return false
  return hasInlineLiveStatus(turn) && turnHasVisibleHost(turn)
}

function hasInlineLiveStatus(turn: Turn): boolean {
  return turn.status === 'inProgress' && runningTurn.value?.id === turn.id
}

function turnHasVisibleHost(turn: Turn): boolean {
  if (finalAgentEntry(turn)) return true
  return turnItemEntries(turn).some((entry) => entry.item.type === 'userMessage' && !isInjectedUserMessage(entry.item))
}

function isEditingFiles(turn: Turn): boolean {
  if (liveChangedFilesForTurn(turn).length > 0) return true
  if (Boolean(turn.diff?.trim()) || turn.items.some((item) => item.type === 'fileChange')) return true
  return turn.items.some((item) => isFileMutationToolItem(item))
}

function isFileMutationToolItem(item: TurnItem): boolean {
  const haystack = `${item.title || ''}\n${item.body || ''}\n${item.auxiliary || ''}\n${JSON.stringify(item.metadata || {})}`.toLowerCase()
  return [
    'apply_patch',
    'update_file',
    'write_file',
    'edit_file',
    'remove-item',
    'new-item',
    'set-content',
    'add-content',
    'git add',
    'git restore',
    'git checkout',
  ].some((needle) => haystack.includes(needle))
}

function formatDurationMs(ms: number): string {
  if (!ms || ms <= 0) return ''
  if (ms < 1000) return `${ms}ms`
  const seconds = Math.round(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const restSeconds = seconds % 60
  if (minutes < 60) return restSeconds > 0 ? `${minutes}m ${restSeconds}s` : `${minutes}m`
  const hours = Math.floor(minutes / 60)
  const restMinutes = minutes % 60
  return restMinutes > 0 ? `${hours}h ${restMinutes}m` : `${hours}h`
}

function diffSummary(diff: string): DiffSummary {
  const cached = diffSummaryCache.get(diff || '')
  if (cached) return cached
  const files: DiffFileSummary[] = []
  let current: DiffFileSummary | null = null
  for (const line of (diff || '').split('\n')) {
    if (line.startsWith('diff --git ')) {
      const match = line.match(/^diff --git a\/(.+?) b\/(.+)$/)
      current = { path: match?.[2] || match?.[1] || 'unknown', additions: 0, deletions: 0 }
      files.push(current)
      continue
    }
    if (line.startsWith('+++ b/') && !current) {
      current = { path: line.slice(6).trim(), additions: 0, deletions: 0 }
      files.push(current)
      continue
    }
    if (!current) continue
    if (line.startsWith('+') && !line.startsWith('+++')) current.additions += 1
    if (line.startsWith('-') && !line.startsWith('---')) current.deletions += 1
  }
  const result = {
    files: filterDisplayChangedFiles(files),
    additions: 0,
    deletions: 0,
  }
  result.additions = result.files.reduce((sum, file) => sum + file.additions, 0)
  result.deletions = result.files.reduce((sum, file) => sum + file.deletions, 0)
  diffSummaryCache.set(diff || '', result)
  return result
}

function turnChangedFiles(turn: Turn): DiffFileSummary[] {
  const byPath = new Map<string, DiffFileSummary>()
  for (const file of diffSummary(turn.diff).files) {
    byPath.set(file.path, { ...file })
  }
  for (const item of turn.items || []) {
    if (item.type === 'userMessage') continue
    if (item.type === 'agentMessage' && !agentMessageHasChangeSummary(item)) continue
    for (const file of fileChangesFromItem(item)) {
      const existing = byPath.get(file.path)
      if (existing) {
        existing.additions = Math.max(existing.additions, file.additions)
        existing.deletions = Math.max(existing.deletions, file.deletions)
      } else {
        byPath.set(file.path, file)
      }
    }
  }
  return filterDisplayChangedFiles(Array.from(byPath.values()))
}

function agentMessageHasChangeSummary(item: TurnItem): boolean {
  const raw = `${item.title || ''}\n${item.body || ''}\n${item.auxiliary || ''}`.toLowerCase()
  return [
    '已编辑',
    '文件已更改',
    '文件变更',
    '代码变更',
    '代码修改',
    '提交:',
    '提交：',
    'commit',
    'git diff',
  ].some((needle) => raw.includes(needle.toLowerCase()))
}

function fileChangesFromItem(item: TurnItem): DiffFileSummary[] {
  const raw = `${item.title || ''}\n${item.body || ''}\n${item.auxiliary || ''}`
  const files: DiffFileSummary[] = []
  const seen = new Set<string>()
  for (const line of raw.split('\n')) {
    const path = extractChangedPathFromLine(line)
    if (!path || seen.has(path)) continue
    seen.add(path)
    const nums = (line.match(/[+-]\d+/g) || []).map((value) => Number(value))
    const additions = Math.max(...nums.filter((value) => value > 0), 0)
    const deletions = Math.abs(Math.min(...nums.filter((value) => value < 0), 0))
    files.push({
      path,
      additions,
      deletions,
    })
  }
  return filterDisplayChangedFiles(files)
}

function extractChangedPathFromLine(line: string): string {
  const trimmed = line.trim()
  if (!trimmed) return ''

  const codexCardMatch = trimmed.match(/^(.+?)\s+([+-]\d+)\s+([+-]\d+)$/)
  if (codexCardMatch) return normalizeChangedPath(codexCardMatch[1])

  const diffMatch = trimmed.match(/^diff --git a\/(.+?) b\/(.+)$/)
  if (diffMatch) return normalizeChangedPath(diffMatch[2] || diffMatch[1])

  const statusMatch = trimmed.match(/^(?:M|A|D|R|C|AM|MM|UU|\?\?)\s+(.+)$/)
  if (statusMatch) return normalizeChangedPath(statusMatch[1])

  const statMatch = trimmed.match(/^(.+?)\s+\|\s+\d+/)
  if (statMatch) return normalizeChangedPath(statMatch[1])

  const genericMatch = trimmed.match(/([A-Za-z0-9_./\\-]+\.(?:go|ts|tsx|js|jsx|vue|css|scss|html|json|md|yaml|yml|toml|rs|py|java|kt|swift|c|cpp|h|hpp|cs|sql))/)
  if (genericMatch) return normalizeChangedPath(genericMatch[1])
  return ''
}

function normalizeChangedPath(path: string): string {
  let value = path.trim().replace(/\\/g, '/')
  value = value.replace(/^"|"$/g, '')
  value = value.replace(/^(\.\.\/)+/, '')
  value = value.replace(/^\.\//, '')
  value = value.replace(/^[ab]\//, '')
  if (!value || value.startsWith('-') || value.includes('://')) return ''
  if (value.includes(' => ')) {
    const parts = value.split(' => ')
    value = parts[parts.length - 1].trim()
  }
  return value
}

function isGeneratedChangePath(path: string): boolean {
  const value = normalizeChangedPath(path)
  return [
    'dist/',
    'dist',
    'web/dist/',
    'web/dist',
    'build/',
    'build',
    'web/build/',
    'web/build',
    'coverage/',
    'coverage',
    'web/coverage/',
    'web/coverage',
    'node_modules/',
    'node_modules',
    'web/node_modules/',
    'web/node_modules',
  ].some((prefix) => value === prefix || value.startsWith(`${prefix}/`))
}

function shouldDisplayChangedFile(file: DiffFileSummary): boolean {
  const path = normalizeChangedPath(file.path)
  if (!path || isGeneratedChangePath(path)) return false
  if (!isProjectRelativeCodePath(path)) return false
  if (!isCodeChangePath(path)) return false
  return file.additions > 0 || file.deletions > 0
}

function isProjectRelativeCodePath(path: string): boolean {
  if (path.startsWith('/') || path.includes('//')) return false
  if (path.includes('/')) return true
  return /^(app|main|index|server|client|vite\.config|webpack\.config|rollup\.config|postcss\.config|tailwind\.config)\./i.test(path)
}

function isCodeChangePath(path: string): boolean {
  return /\.(go|tsx?|jsx?|mjs|cjs|vue|css|scss|sass|less|html|rs|py|java|kt|swift|c|cc|cpp|h|hpp|cs|sql|sh|ps1|bat|cmd|svelte)$/i.test(path)
}

function filterDisplayChangedFiles(files: DiffFileSummary[]): DiffFileSummary[] {
  const byPath = new Map<string, DiffFileSummary>()
  for (const file of files) {
    const path = normalizeChangedPath(file.path)
    const normalized = { ...file, path }
    if (!shouldDisplayChangedFile(normalized)) continue
    const existing = byPath.get(path)
    if (existing) {
      existing.additions = Math.max(existing.additions, normalized.additions)
      existing.deletions = Math.max(existing.deletions, normalized.deletions)
    } else {
      byPath.set(path, normalized)
    }
  }
  return Array.from(byPath.values())
}

function visibleTurnChangedFiles(turn: Turn): DiffFileSummary[] {
  const files = turnDisplayChangedFiles(turn)
  if (expandedTurnChangeIds.value.has(turn.id)) return files
  return files.slice(0, turnChangePreviewLimit)
}

function hiddenTurnChangeCount(turn: Turn): number {
  if (expandedTurnChangeIds.value.has(turn.id)) return 0
  return Math.max(turnDisplayChangedFiles(turn).length - turnChangePreviewLimit, 0)
}

function toggleTurnChangeExpanded(turnID: string) {
  const next = new Set(expandedTurnChangeIds.value)
  if (next.has(turnID)) next.delete(turnID)
  else next.add(turnID)
  expandedTurnChangeIds.value = next
}

function diffLines(diff: string): string[] {
  return (diff || '').split('\n').filter((line) => line.length > 0)
}

function diffLineClass(line: string) {
  if (line.startsWith('+++') || line.startsWith('---')) return 'is-meta'
  if (line.startsWith('+')) return 'is-add'
  if (line.startsWith('-')) return 'is-del'
  if (line.startsWith('@@')) return 'is-hunk'
  if (line.startsWith('diff --git')) return 'is-file'
  return ''
}

function renderMarkdown(source: string): string {
  return normalizeAttachedImageSyntax(rewriteMarkdownImagePaths(source || ''))
}

function messageImages(source: string): MessageImage[] {
  const token = localStorage.getItem('cf_token') || ''
  const images: MessageImage[] = []
  const seen = new Set<string>()
  const addImage = (alt: string, rawPath: string) => {
    const normalizedPath = normalizeImagePath(rawPath)
    if (!normalizedPath) return
    const url = buildLocalImageUrl(normalizedPath, token)
    if (seen.has(url)) return
    seen.add(url)
    images.push({ url, alt: alt || 'image', index: images.length })
  }

  for (const match of (source || '').matchAll(/!\[([^\]]*)\]\(([^)]+)\)/g)) {
    addImage(match[1] || '', match[2] || '')
  }
  for (const match of (source || '').matchAll(/\[Attached image:\s*([^\]]+?)\]/g)) {
    addImage('Attached image', match[1] || '')
  }
  return images
}

function messagePreviewUrls(source: string): string[] {
  return messageImages(source).map((image) => image.url)
}

function messageText(source: string): string {
  return (source || '')
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '')
    .replace(/\[Attached image:\s*([^\]]+?)\]/g, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

function itemImages(item: TurnItem): MessageImage[] {
  return messageImages(item.body || '')
}

function itemPreviewUrls(item: TurnItem): string[] {
  return messagePreviewUrls(item.body || '')
}

function itemText(item: TurnItem): string {
  const text = messageText(item.body || '')
  if (item.type !== 'userMessage') return text
  return stripBrowserPromptScaffold(text)
}

function stripBrowserPromptScaffold(source: string): string {
  const marker = '## My request for Codex:'
  const markerIndex = source.indexOf(marker)
  if (markerIndex >= 0) {
    return source.slice(markerIndex + marker.length).trim()
  }
  return source
    .replace(/^# In app browser:\s*(?:\n- .*)+\n*/i, '')
    .trim()
}

function rewriteMarkdownImagePaths(source: string): string {
  const token = localStorage.getItem('cf_token') || ''
  return source.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_full, alt: string, rawPath: string) => {
    const normalizedPath = normalizeImagePath(rawPath)
    if (!normalizedPath) return `![${alt}](${rawPath})`
    return `![${alt}](${buildLocalImageUrl(normalizedPath, token)})`
  })
}

function normalizeAttachedImageSyntax(source: string): string {
  const token = localStorage.getItem('cf_token') || ''
  return source.replace(/\[Attached image:\s*([^\]]+?)\]/g, (_full, rawPath: string) => {
    const normalizedPath = normalizeImagePath(rawPath)
    if (!normalizedPath) return _full
    return `\n\n![Attached image](${buildLocalImageUrl(normalizedPath, token)})\n\n`
  })
}

function normalizeImagePath(rawPath: string): string {
  const trimmed = rawPath.trim().replace(/^<|>$/g, '').replace(/^['"]|['"]$/g, '')
  if (!trimmed) return ''
  if (/^(https?:)?\/\//i.test(trimmed)) return trimmed
  if (/^(data:image\/)/i.test(trimmed)) return trimmed
  if (/^(inline:)/i.test(trimmed)) return trimmed
  if (/^(upload:)/i.test(trimmed)) return trimmed
  if (/^[A-Za-z]:[\\/]/.test(trimmed) || trimmed.startsWith('/')) return trimmed
  return ''
}

function buildLocalImageUrl(path: string, token: string): string {
  if (/^(https?:)?\/\//i.test(path) || /^(data:image\/)/i.test(path)) {
    return path
  }
  const params = new URLSearchParams({ path })
  if (token) params.set('token', token)
  return `${localAssetBase}?${params.toString()}`
}

function isStreamingItem(turn: Turn, item: TurnItem, index: number): boolean {
  if (turn.status !== 'inProgress' || item.type !== 'agentMessage') return false
  for (let i = turn.items.length - 1; i > index; i -= 1) {
    const next = turn.items[i]
    if (!next) continue
    if (next.type !== 'reasoning' && next.type !== 'plan') return false
    if (next.body?.trim() || next.auxiliary?.trim() || next.status?.trim()) return false
  }
  return true
}

function turnNumber(id: string) {
  const idx = orderedTurns.value.findIndex((turn) => turn.id === id)
  return idx >= 0 ? idx + 1 : '?'
}

function setChatScrollToBottom(force = false) {
  const el = chatAreaRef.value
  if (!el) {
    if (force || followLiveOutput.value) window.scrollTo({ top: document.documentElement.scrollHeight })
    return
  }
  if (!force && !followLiveOutput.value) return
  el.scrollTop = el.scrollHeight
  if (isMobile.value) {
    window.scrollTo({ top: document.documentElement.scrollHeight })
    document.scrollingElement?.scrollTo({ top: document.scrollingElement.scrollHeight })
  }
}

function scrollChatToBottom(force = false) {
  nextTick(() => {
    setChatScrollToBottom(force)
  })
}

async function scrollChatToBottomAfterLayout(force = false) {
  await nextTick()
  setChatScrollToBottom(force)
  requestAnimationFrame(() => {
    setChatScrollToBottom(force)
    requestAnimationFrame(() => setChatScrollToBottom(force))
  })
  window.setTimeout(() => setChatScrollToBottom(force), 120)
  window.setTimeout(() => setChatScrollToBottom(force), 360)
  window.setTimeout(() => setChatScrollToBottom(force), 800)
}

function handleMessageAssetLoad() {
  if (followLiveOutput.value || !initialScrollDone) {
    scrollChatToBottom(true)
  }
}

function handleMessageAssetError() {
  handleMessageAssetLoad()
}

function scrollInputIntoView() {
  nextTick(() => {
    const el = chatAreaRef.value
    if (!el) return
    const input = document.querySelector('.session-detail-page .input-area')
    input?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
}

function jumpToLatest() {
  followLiveOutput.value = true
  pendingNewMessages.value = 0
  scrollChatToBottomAfterLayout(true)
}

async function onChatScroll() {
  const el = chatAreaRef.value
  if (!el) return
  const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
  followLiveOutput.value = nearBottom
  if (nearBottom) {
    pendingNewMessages.value = 0
  }
  if (el.scrollTop < 40 && detail.value?.hasMoreHistory && !loadingHistory.value) {
    await loadOlderTurns()
  }
}

watch(orderedTurns, (next, prev) => {
  const prevLast = prev?.[prev.length - 1]
  const nextLast = next?.[next.length - 1]
  const latestChanged = !prevLast || !nextLast || prevLast.id !== nextLast.id || JSON.stringify(prevLast.items) !== JSON.stringify(nextLast.items)
  if (latestChanged && followLiveOutput.value) {
    scrollChatToBottomAfterLayout(true)
    pendingNewMessages.value = 0
  } else if (latestChanged) {
    pendingNewMessages.value += 1
  }
  refreshLiveChanges()
}, { deep: true })

async function refreshPage() {
  await app.refreshDashboard()
  await app.loadSession(sessionId)
}

function changeQuery(scope = changeScope.value) {
  return {
    scope,
    ref: scope === 'commit' ? changeRef.value.trim() : '',
    base: scope === 'base' ? changeBase.value.trim() : '',
    turnId: scope === 'turn' ? activeTurnChangeId.value : '',
  }
}

function reviewQuery(scope = reviewScope.value) {
  return {
    scope,
    ref: scope === 'commit' ? reviewRef.value.trim() : '',
    base: scope === 'base' ? reviewBase.value.trim() : '',
    turnId: scope === 'turn' ? activeReviewTurnId.value : '',
  }
}

async function openChangesDrawer() {
  activeTurnChangeId.value = ''
  changeScope.value = 'workspace'
  changesDrawerOpen.value = true
  await reloadChanges()
}

async function reloadChanges() {
  changesLoading.value = true
  changesError.value = ''
  selectedFileDetail.value = null
  selectedChangeFile.value = ''
  try {
    const data = await app.loadSessionChanges(sessionId, changeQuery())
    changes.value = data
    const firstFile = data.files?.[0]
    if (firstFile) {
      await selectChangedFile(firstFile.path)
    }
  } catch (e: any) {
    changes.value = null
    changesError.value = e.response?.data?.error || '读取变更失败'
  } finally {
    changesLoading.value = false
  }
}

async function selectChangedFile(path: string) {
  if (!path) return
  selectedChangeFile.value = path
  fileViewMode.value = 'diff'
  try {
    const data = await app.loadSessionChanges(sessionId, { ...changeQuery(), file: path })
    selectedFileDetail.value = data.file || null
  } catch (e: any) {
    selectedFileDetail.value = null
    ElMessage.error(e.response?.data?.error || '读取文件失败')
  }
}

async function openTurnChangedFile(turn: Turn, path: string) {
  activeTurnChangeId.value = turn.id
  changeScope.value = 'turn'
  changesDrawerOpen.value = true
  await reloadChanges()
  if (path) {
    await selectChangedFile(path)
  }
}

function reviewTurnChanges(turn: Turn) {
  activeReviewTurnId.value = turn.id
  reviewScope.value = 'turn'
  reviewRef.value = ''
  reviewBase.value = changeBase.value || 'main'
  reviewDialogOpen.value = true
  reloadReviewPreview()
}

async function revertTurnChanges(turn: Turn) {
  const files = turnChangedFiles(turn).map((file) => file.path)
  if (files.length === 0) return
  try {
    await ElMessageBox.confirm(
      `将撤销 ${files.length} 个工作区文件的未提交改动。未跟踪文件会被删除，这个操作无法在 CodexPocket 内撤回。`,
      '撤销文件改动',
      {
        confirmButtonText: '撤销改动',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )
  } catch {
    return
  }

  revertingFiles.value = true
  try {
    const data = await app.revertSessionChanges(sessionId, files)
    changes.value = data
    selectedFileDetail.value = null
    selectedChangeFile.value = ''
    ElMessage.success('已撤销文件改动')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '撤销失败')
  } finally {
    revertingFiles.value = false
  }
}

function openReviewDialog() {
  activeReviewTurnId.value = ''
  reviewScope.value = changeScope.value
  if (reviewScope.value === 'turn') reviewScope.value = 'workspace'
  reviewRef.value = changeRef.value
  reviewBase.value = changeBase.value
  reviewDialogOpen.value = true
  reloadReviewPreview()
}

async function reloadReviewPreview() {
  reviewPreviewLoading.value = true
  reviewPreviewError.value = ''
  selectedReviewFile.value = ''
  selectedReviewFileDetail.value = null
  try {
    const data = await app.loadSessionChanges(sessionId, reviewQuery())
    data.files = data.files.filter((file) => shouldDisplayChangedFile(file))
    data.summary.files = data.files.length
    data.summary.additions = data.files.reduce((sum, file) => sum + file.additions, 0)
    data.summary.deletions = data.files.reduce((sum, file) => sum + file.deletions, 0)
    reviewPreview.value = data
    const firstFile = data.files?.[0]
    if (firstFile) {
      await selectReviewFile(firstFile.path)
    }
  } catch (e: any) {
    reviewPreview.value = null
    reviewPreviewError.value = e.response?.data?.error || '读取审查内容失败'
  } finally {
    reviewPreviewLoading.value = false
  }
}

async function selectReviewFile(path: string) {
  if (!path) return
  selectedReviewFile.value = path
  try {
    const data = await app.loadSessionChanges(sessionId, { ...reviewQuery(), file: path })
    selectedReviewFileDetail.value = data.file || null
  } catch (e: any) {
    selectedReviewFileDetail.value = null
    ElMessage.error(e.response?.data?.error || '读取 diff 失败')
  }
}

async function handleStartReview() {
  if (reviewScope.value === 'commit' && !reviewRef.value.trim()) {
    ElMessage.warning('请填写 commit hash')
    return
  }
  if (reviewScope.value === 'base' && !reviewBase.value.trim()) {
    ElMessage.warning('请填写 base branch')
    return
  }
  if (!summary.value?.loaded) {
    try {
      await ElMessageBox.confirm('Review 会作为新的 Codex turn 发送，需要先接管这个会话。', '接管后审查', {
        confirmButtonText: '接管并继续',
        cancelButtonText: '取消',
        type: 'info',
      })
      await app.resumeSession(sessionId)
    } catch {
      return
    }
  }

  reviewing.value = true
  try {
    await app.startReview(sessionId, {
      scope: reviewScope.value,
      ref: reviewScope.value === 'commit' ? reviewRef.value.trim() : '',
      base: reviewScope.value === 'base' ? reviewBase.value.trim() : '',
      turnId: reviewScope.value === 'turn' ? activeReviewTurnId.value : '',
    })
    reviewDialogOpen.value = false
    followLiveOutput.value = true
    ElMessage.success('已开始审查改动')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '审查启动失败')
  } finally {
    reviewing.value = false
  }
}

async function refreshSessionWhenVisible() {
  if (document.visibilityState !== 'visible') return
  await app.loadSession(sessionId)
}

function tryClaimLiveLease() {
  const now = Date.now()
  try {
    const raw = localStorage.getItem(liveLeaseKey)
    const lease = raw ? JSON.parse(raw) : null
    if (lease?.owner && lease.owner !== tabInstanceId && Number(lease.expiresAt || 0) > now) {
      return false
    }
    localStorage.setItem(liveLeaseKey, JSON.stringify({
      owner: tabInstanceId,
      expiresAt: now + liveLeaseMs,
    }))
    return true
  } catch {
    return true
  }
}

function publishLiveSnapshot() {
  const current = detail.value
  if (!current) return
  try {
    localStorage.setItem(liveSnapshotKey, JSON.stringify({
      owner: tabInstanceId,
      updatedAt: Date.now(),
      detail: current,
    }))
  } catch { /* storage unavailable */ }
}

async function syncLiveTranscript() {
  if (document.visibilityState !== 'visible' || liveSyncBusy) return
  if (summary.value?.agentId && summary.value.agentId !== 'codex') return
  if (!tryClaimLiveLease()) return
  liveSyncBusy = true
  try {
    await app.loadSession(sessionId, { fast: true })
    await refreshLiveChanges()
    publishLiveSnapshot()
  } finally {
    liveSyncBusy = false
  }
}

async function refreshLiveChanges() {
  if (!runningTurn.value || liveChangesBusy) {
    if (!runningTurn.value) liveChanges.value = null
    return
  }
  liveChangesBusy = true
  try {
    liveChanges.value = await app.loadSessionChanges(sessionId, { scope: 'workspace' })
  } catch {
    liveChanges.value = null
  } finally {
    liveChangesBusy = false
  }
}

function onLiveStorage(event: StorageEvent) {
  if (event.key !== liveSnapshotKey || !event.newValue) return
  try {
    const payload = JSON.parse(event.newValue)
    if (payload?.owner === tabInstanceId || !payload?.detail) return
    app.replaceSessionDetail(sessionId, payload.detail)
  } catch { /* ignore stale snapshot */ }
}

async function loadOlderTurns() {
  if (!detail.value?.hasMoreHistory || loadingHistory.value) return
  loadingHistory.value = true
  const el = chatAreaRef.value
  const beforeHeight = el?.scrollHeight || 0
  const beforeTop = el?.scrollTop || 0
  try {
    const nextOffset = Math.max((detail.value.offset || 0) - (detail.value.limit || 8), 0)
    await app.loadSession(sessionId, {
      offset: nextOffset,
      limit: detail.value.limit || 8,
      appendHistory: true,
    })
    await nextTick()
    if (el) {
      const delta = el.scrollHeight - beforeHeight
      el.scrollTop = beforeTop + delta
    }
  } finally {
    loadingHistory.value = false
  }
}

function onAction(cmd: string) {
  if (cmd === 'resume') handleResume()
  else if (cmd === 'detach') handleDetach()
  else if (cmd === 'end') handleEnd()
  else if (cmd === 'rename') handleRename()
  else if (cmd === 'goal') handleGoal()
  else if (cmd === 'goal-clear') handleClearGoal()
  else if (cmd === 'fork') handleFork()
  else if (cmd === 'compact') handleCompact()
  else if (cmd === 'rollback') handleRollback()
}

async function handleResume() {
  resuming.value = true
  try {
    await app.resumeSession(sessionId)
    ElMessage.success('会话已接管')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '接管失败')
  } finally {
    resuming.value = false
  }
}

async function handleSubmit() {
  if (!promptText.value.trim() && pendingImages.value.length === 0) return
  submitting.value = true
  try {
    const text = promptText.value
    const imageIds = pendingImages.value.map((image) => image.id)
    const autoResume = !summary.value?.loaded
    if (autoResume) {
      resuming.value = true
      await app.resumeSession(sessionId)
      resuming.value = false
    }
    const activeTurn = runningTurn.value
    if (activeTurn?.id) {
      await app.steerTurn(sessionId, activeTurn.id, text, imageIds)
    } else {
      await app.startTurn(sessionId, text, imageIds)
    }
    promptText.value = ''
    pendingImages.value = []
    followLiveOutput.value = true
    ElMessage.success(autoResume ? '已接管并发送' : '指令已发送')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '发送失败')
  } finally {
    resuming.value = false
    submitting.value = false
  }
}

function onInputAction(command: string) {
  if (command === 'image') {
    imageInputRef.value?.click()
  } else if (command === 'changes') {
    openChangesDrawer()
  } else if (command === 'review') {
    openReviewDialog()
  }
}

async function handleImageFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (files.length === 0) return
  uploadingImage.value = true
  try {
    for (const file of files) {
      const data = new FormData()
      data.append('file', file)
      const res = await api.post('/uploads/image', data, { timeout: 60000 })
      pendingImages.value.push({
        id: res.data.id,
        name: res.data.name || file.name,
        size: res.data.size || file.size,
      })
    }
    ElMessage.success(files.length > 1 ? `已添加 ${files.length} 张图片` : '图片已添加')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '图片上传失败')
  } finally {
    uploadingImage.value = false
  }
}

function removePendingImage(id: string) {
  pendingImages.value = pendingImages.value.filter((image) => image.id !== id)
}

async function handleDetach() {
  detaching.value = true
  try {
    await ElMessageBox.confirm('确定要取消接管这个会话吗？', '确认')
    await app.detachSession(sessionId)
    ElMessage.success('已取消接管')
  } catch { /* cancelled */ }
  finally {
    detaching.value = false
  }
}

async function handleInterrupt() {
  const turnId = runningTurn.value?.id || summary.value?.lastTurnId
  if (!turnId) return
  try {
    await app.interruptTurn(sessionId, turnId)
    ElMessage.success('已中断')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '中断失败')
  }
}

async function handleEnd() {
  try {
    await ElMessageBox.confirm('确定要结束这个会话吗？', '确认')
    await app.endSession(sessionId)
    ElMessage.success('会话已结束')
  } catch { /* cancelled */ }
}

async function handleRename() {
  const currentName = summary.value ? displayName(summary.value) : ''
  try {
    const { value } = await ElMessageBox.prompt('给这个会话起一个更容易识别的名字', '重命名会话', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputValue: currentName,
      inputPattern: /\S+/,
      inputErrorMessage: '名称不能为空',
    })
    const name = String(value || '').trim()
    if (!name) return
    await app.renameSession(sessionId, name)
    ElMessage.success('会话已重命名')
  } catch { /* cancelled */ }
}

async function handleGoal() {
  const current = detail.value?.goal?.objective || ''
  try {
    const { value } = await ElMessageBox.prompt('设置这个会话的目标，Codex 会把它作为持续任务目标保存。', '设置目标', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputType: 'textarea',
      inputValue: current,
      inputPattern: /\S+/,
      inputErrorMessage: '目标不能为空',
    })
    const objective = String(value || '').trim()
    if (!objective) return
    await app.setSessionGoal(sessionId, objective)
    ElMessage.success('目标已保存')
  } catch { /* cancelled */ }
}

async function handleClearGoal() {
  try {
    await ElMessageBox.confirm('确定要清空当前会话目标吗？', '清空目标', {
      confirmButtonText: '清空',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await app.clearSessionGoal(sessionId)
    ElMessage.success('目标已清空')
  } catch { /* cancelled */ }
}

async function handleFork() {
  try {
    await ElMessageBox.confirm('会基于当前历史创建一个新的 Codex 会话分支。', '分支会话', {
      confirmButtonText: '创建分支',
      cancelButtonText: '取消',
      type: 'info',
    })
    const forked = await app.forkSession(sessionId)
    ElMessage.success('分支会话已创建')
    router.push(`/session/${forked.id}`)
  } catch { /* cancelled */ }
}

async function handleCompact() {
  try {
    await ElMessageBox.confirm('Codex 会开始压缩当前会话上下文，过程会作为新的消息流显示。', '压缩上下文', {
      confirmButtonText: '开始压缩',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await app.compactSession(sessionId)
    ElMessage.success('已开始压缩上下文')
  } catch { /* cancelled */ }
}

async function handleRollback() {
  try {
    await ElMessageBox.confirm('会从 Codex 上下文中移除最近 1 轮，并写入回滚记录。这个操作无法在 CodexPocket 内撤销。', '回滚最近一轮', {
      confirmButtonText: '回滚',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await app.rollbackSession(sessionId, 1)
    ElMessage.success('已回滚最近一轮')
    await nextTick()
    scrollChatToBottom(true)
  } catch { /* cancelled */ }
}

function approvalChoices(approval: ApprovalRequest) {
  if (approval.kind === 'userInput') {
    return [{ value: 'answer', label: '回复', type: 'primary' }]
  }
  const choices = approval.choices?.length ? approval.choices : ['accept', 'decline']
  return choices.map((choice) => ({
    value: choice,
    label: choiceLabel(choice),
    type: choiceType(choice),
  }))
}

function choiceLabel(choice: string) {
  switch (choice) {
    case 'accept': return '批准本次'
    case 'acceptForSession': return '本会话批准'
    case 'decline': return '拒绝'
    case 'deny': return '拒绝'
    case 'cancel': return '取消'
    case 'session': return '允许本会话'
    case 'turn': return '允许本轮'
    case 'answer': return '回复'
    default: return choice
  }
}

function choiceType(choice: string) {
  switch (choice) {
    case 'accept':
    case 'acceptForSession':
    case 'session':
    case 'turn':
      return 'success'
    case 'decline':
    case 'deny':
    case 'cancel':
      return 'danger'
    default:
      return 'primary'
  }
}

function goalStatusLabel(status: string) {
  switch (status) {
    case 'active': return '进行中'
    case 'complete': return '已完成'
    case 'blocked': return '已阻塞'
    default: return status || '目标'
  }
}

function formatGoalTime(seconds: number) {
  if (!seconds || seconds <= 0) return ''
  const minutes = Math.floor(seconds / 60)
  if (minutes < 1) return `${seconds}s`
  const hours = Math.floor(minutes / 60)
  if (hours < 1) return `${minutes}m`
  return `${hours}h ${minutes % 60}m`
}

async function handleApprovalChoice(approval: ApprovalRequest, decision: string) {
  try {
    let result: Record<string, any>
    if (approval.kind === 'command' || approval.kind === 'fileChange' || approval.kind === 'generic') {
      result = { decision }
    } else if (approval.kind === 'permissions') {
      result = decision === 'session' || decision === 'turn'
        ? { permissions: approval.params?.permissions || {}, scope: decision }
        : { permissions: null, scope: null }
    } else if (approval.kind === 'userInput') {
      const { value } = await ElMessageBox.prompt('请输入回复', '用户输入', {
        confirmButtonText: '提交',
        cancelButtonText: '取消',
      })
      const questionId = approval.params?.questions?.[0]?.id || 'reply'
      result = { answers: { [questionId]: { answers: [value] } } }
    } else {
      result = { decision }
    }
    await app.resolveApproval(approval.id, result)
    ElMessage.success('审批已提交')
  } catch { /* cancelled */ }
}

onMounted(async () => {
  await refreshPage()
  await refreshLiveChanges()
  app.registerActiveSession(sessionId)
  document.addEventListener('visibilitychange', refreshSessionWhenVisible)
  window.addEventListener('focus', refreshSessionWhenVisible)
  window.addEventListener('storage', onLiveStorage)
  liveSyncTimer = setInterval(syncLiveTranscript, liveSyncIntervalMs)
  liveChangesTimer = setInterval(refreshLiveChanges, liveSyncIntervalMs)
  elapsedTimer = setInterval(() => {
    elapsedNow.value = Date.now()
  }, 1000)
  await scrollChatToBottomAfterLayout(true)
  initialScrollDone = true
})

watch(() => summary.value?.loaded, (next, prev) => {
  if (!next || !isMobile.value || prev === undefined || next === prev) return
  scrollInputIntoView()
})

onUnmounted(() => {
  app.unregisterActiveSession(sessionId)
  if (liveSyncTimer) {
    clearInterval(liveSyncTimer)
    liveSyncTimer = null
  }
  if (liveChangesTimer) {
    clearInterval(liveChangesTimer)
    liveChangesTimer = null
  }
  if (elapsedTimer) {
    clearInterval(elapsedTimer)
    elapsedTimer = null
  }
  try {
    const raw = localStorage.getItem(liveLeaseKey)
    const lease = raw ? JSON.parse(raw) : null
    if (lease?.owner === tabInstanceId) localStorage.removeItem(liveLeaseKey)
  } catch { /* ignore */ }
  document.removeEventListener('visibilitychange', refreshSessionWhenVisible)
  window.removeEventListener('focus', refreshSessionWhenVisible)
  window.removeEventListener('storage', onLiveStorage)
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
.session-detail-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  margin: 0;
  overflow: hidden;
  min-height: 0;
}

.input-area {
  flex-shrink: 0;
}

.goal-card {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin: 0 18px 8px;
  padding: 10px 14px;
  border: 1px solid rgba(151, 194, 255, 0.75);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.86);
  box-shadow: var(--cf-shadow-sm);
}

.goal-main {
  min-width: 0;
  flex: 1;
}

.goal-label {
  margin-bottom: 4px;
  font-size: 12px;
  font-weight: 700;
  color: var(--cf-primary);
}

.goal-objective {
  color: var(--cf-text-heavy);
  font-weight: 700;
  line-height: 1.5;
  word-break: break-word;
}

.goal-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 6px;
  color: var(--cf-text-secondary);
  font-size: 12px;
}

.goal-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.session-hero {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px 12px;
  margin: 0 18px 5px;
  border: 1px solid rgba(205, 223, 255, 0.78);
  border-radius: 12px;
  background:
    linear-gradient(140deg, rgba(51, 136, 255, 0.1) 0%, rgba(51, 136, 255, 0.02) 46%, rgba(255, 255, 255, 0.96) 100%),
    #fff;
  box-shadow: var(--cf-shadow-sm);
}

.hero-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.back-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 0;
  border-radius: 999px;
  min-height: 26px;
  padding: 4px 9px;
  background: rgba(255, 255, 255, 0.88);
  color: var(--cf-text-secondary);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: inset 0 0 0 1px rgba(205, 223, 255, 0.8);
}

.back-chip:hover {
  color: var(--cf-primary-dark);
  box-shadow: inset 0 0 0 1px rgba(121, 168, 255, 0.95);
}

.hero-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.hero-main {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
}

.hero-title-group {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.hero-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  min-width: 0;
}

.hero-name {
  min-width: 0;
  font-size: 18px;
  line-height: 1.1;
  font-weight: 700;
  color: var(--cf-text-heavy);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hero-meta-row {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  flex-wrap: wrap;
}

.hero-cwd {
  font-size: 11px;
  color: var(--cf-text-secondary);
  font-family: monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
  order: 1;
}

.hero-tags {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 5px;
  flex-wrap: wrap;
  order: 2;
}

.hero-pill {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  min-height: 20px;
  padding: 0 7px;
  border-radius: 999px;
  background: rgba(51, 136, 255, 0.08);
  color: var(--cf-primary-dark);
  font-size: 11px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hero-pill.is-active {
  background: rgba(19, 168, 107, 0.12);
  color: var(--cf-success);
}

.hero-preview {
  margin: 0;
  font-size: 11px;
  line-height: 1.35;
  color: var(--cf-text-secondary);
  max-width: 780px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.hero-status-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 300px;
  padding: 6px 8px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid rgba(216, 230, 251, 0.95);
  box-shadow: 0 6px 14px rgba(15, 46, 106, 0.05);
}

.hero-status-copy {
  min-width: 0;
  flex: 1;
}

.hero-status-label {
  display: none;
  color: var(--cf-text-lighter);
  font-weight: 600;
}

.hero-status-value {
  font-size: 12px;
  line-height: 1.2;
  font-weight: 700;
  color: var(--cf-text-heavy);
  margin-top: 1px;
}

.hero-status-desc {
  display: none;
  font-size: 11px;
  line-height: 1.25;
  color: var(--cf-text-secondary);
  max-width: 150px;
}

.hero-primary-actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  flex-shrink: 0;
}

.hero-primary-actions :deep(.el-button) {
  min-width: 92px;
  min-height: 28px;
  border-radius: 8px;
}

.hero-actions :deep(.el-button) {
  border-radius: 10px;
}

.live-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--cf-warning);
}

.live-indicator {
  padding: 2px 7px;
  border-radius: 999px;
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.live-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--cf-warning);
  animation: live-pulse 1.5s ease-in-out infinite;
}

@keyframes live-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.8); }
}

.session-meta {
  padding: 8px 16px;
  background: var(--cf-card);
  border-bottom: 1px solid var(--cf-border-light);
  cursor: pointer;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.meta-cwd {
  font-size: 12px;
  color: var(--cf-text-secondary);
  font-family: monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 420px;
}

.meta-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.meta-arrow {
  margin-left: auto;
  transition: transform 0.2s ease;
  color: var(--cf-text-lighter);
  font-size: 12px;
}

.meta-arrow.is-up {
  transform: rotate(90deg);
}

.meta-preview {
  font-size: 12px;
  color: var(--cf-text-secondary);
  margin-top: 6px;
  line-height: 1.5;
}

.resume-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #fffbeb;
  border-bottom: 1px solid #fde68a;
  font-size: 13px;
  color: #92400e;
}

.content-area {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 18px 0;
  background: linear-gradient(180deg, #eef5fd 0%, #e7f0fb 100%);
}

.approval-section {
  flex-shrink: 0;
  padding: 0 0 10px;
}

.approval-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--cf-card);
  border-radius: 10px;
  border-left: 3px solid var(--cf-warning);
  margin-bottom: 6px;
  gap: 8px;
}

.approval-info {
  min-width: 0;
  flex: 1;
}

.approval-kind {
  font-size: 13px;
  font-weight: 600;
}

.approval-reason {
  font-size: 12px;
  color: var(--cf-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.approval-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.chat-shell {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: rgba(248, 251, 255, 0.92);
  border: 1px solid #dce8f8;
  border-radius: 20px 20px 0 0;
  overflow: hidden;
  position: relative;
}

.chat-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px 10px;
  border-bottom: 1px solid rgba(220, 230, 246, 0.9);
  background: rgba(255, 255, 255, 0.85);
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.follow-tip {
  font-size: 12px;
  color: var(--cf-text-secondary);
}

.chat-area {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 14px 18px 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.history-load-row,
.empty-hint {
  display: flex;
  justify-content: center;
}

.history-load-row {
  min-height: 24px;
}

.empty-hint {
  align-items: center;
  gap: 8px;
  padding: 40px 0;
  color: var(--cf-text-secondary);
  font-size: 14px;
}

.turn-stream {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.turn-anchor {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 12px;
  color: var(--cf-text-lighter);
}

.turn-title {
  font-weight: 600;
  color: var(--cf-text-secondary);
}

.activity-row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  align-self: flex-start;
  width: min(100%, 860px);
  padding: 10px 14px;
  border: 1px solid rgba(216, 230, 251, 0.9);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.72);
  color: var(--cf-text-secondary);
  font-size: 13px;
  font-weight: 600;
}

.activity-row.is-latest {
  border-color: rgba(245, 158, 11, 0.32);
  background: rgba(255, 251, 235, 0.86);
  color: #b45309;
}

.activity-row.is-latest .activity-spinner {
  border-color: rgba(245, 158, 11, 0.24);
  border-top-color: var(--cf-warning);
}

.activity-spinner {
  width: 12px;
  height: 12px;
  flex: 0 0 auto;
  border-radius: 50%;
  border: 2px solid rgba(51, 136, 255, 0.22);
  border-top-color: var(--cf-primary);
  animation: activity-spin 0.85s linear infinite;
}

.activity-spinner.is-small {
  width: 10px;
  height: 10px;
  border-width: 1.5px;
}

@keyframes activity-spin {
  to { transform: rotate(360deg); }
}

.turn-process {
  width: min(100%, 860px);
  margin-left: 0;
  border: 1px solid #d9e6f7;
  border-radius: 14px;
  background: rgba(248, 251, 255, 0.78);
  box-shadow: 0 8px 18px rgba(15, 46, 106, 0.035);
}

.turn-process.is-compact {
  border-color: rgba(226, 232, 240, 0.95);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.62);
  box-shadow: none;
}

.turn-process.is-inline {
  width: 100%;
  margin: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.turn-process.is-inline .turn-process-summary {
  padding: 0 0 8px;
}

.turn-process.is-inline .turn-process-items {
  padding: 8px 0 0;
  border-top: 1px solid rgba(226, 232, 240, 0.95);
}

.process-summary-divider {
  height: 1px;
  margin: 10px 0 12px;
  background: rgba(216, 230, 251, 0.95);
}

.turn-process-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 12px;
  cursor: pointer;
  color: var(--cf-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.turn-process-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.turn-process-duration {
  color: var(--cf-text-lighter);
  font-weight: 600;
  white-space: nowrap;
}

.turn-process-items {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0 12px 12px;
  border-top: 1px solid rgba(216, 230, 251, 0.8);
}

.process-entry-card,
.process-command-group {
  border: 0;
  border-radius: 0;
  background: transparent;
  padding: 0;
}

.process-entry-card.is-agentMessage {
  color: var(--cf-text-heavy);
}

.process-entry-head,
.process-command-summary {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
  color: var(--cf-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.process-command-summary {
  cursor: pointer;
  width: fit-content;
}

.process-command-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 8px;
}

.process-command-list .process-entry-card {
  padding: 8px 0 0 18px;
  border-left: 1px solid rgba(226, 232, 240, 0.95);
}

.process-entry-card.is-live-file-edit {
  color: var(--cf-text-secondary);
}

.file-edit-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 6px;
}

.file-edit-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  min-height: 24px;
  color: var(--cf-text-secondary);
  font-size: 12px;
}

.process-live-row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--cf-text-lighter);
  font-size: 12px;
  font-weight: 600;
}

.file-edit-path {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--cf-primary-dark);
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
}

.file-edit-stats {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  white-space: nowrap;
  font-size: 12px;
}

.turn-process-items .message-bubble {
  width: 100%;
  box-shadow: none;
}

.message-row {
  display: flex;
  width: 100%;
}

.message-row.side-left {
  justify-content: flex-start;
}

.message-row.side-right {
  justify-content: flex-end;
}

.message-bubble {
  width: min(100%, 860px);
  border-radius: 18px;
  padding: 12px 14px;
  box-shadow: 0 10px 24px rgba(15, 46, 106, 0.04);
  border: 1px solid transparent;
}

.bubble-user {
  max-width: min(78%, 760px);
  background: #2f6fec;
  color: #fff;
  border-color: #2f6fec;
}

.bubble-user :deep(*) {
  color: #fff;
}

.bubble-agent {
  background: #ffffff;
  border-color: #d8e6fb;
}

.bubble-tool {
  background: #f8fbff;
  border-color: #d9e6f7;
}

.bubble-meta {
  background: #f7fafc;
  border-color: #e5ebf5;
}

.bubble-other {
  background: #ffffff;
  border-color: #e5e7eb;
}

.bubble-error {
  background: #fff5f5;
  border-color: #fecaca;
}

.message-topline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 6px;
}

.message-label {
  font-size: 12px;
  font-weight: 700;
}

.message-status {
  font-size: 11px;
  opacity: 0.7;
}

.message-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 6px;
}

.message-body {
  font-size: 14px;
  line-height: 1.65;
  color: var(--cf-text-secondary);
}

.bubble-user .message-body {
  color: #fff;
}

.image-strip {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: 100%;
  margin-bottom: 10px;
  overflow-x: auto;
  padding: 2px 2px 4px;
}

.image-strip:last-child {
  margin-bottom: 0;
}

.message-thumb {
  width: 78px;
  height: 78px;
  flex: 0 0 auto;
  overflow: hidden;
  border-radius: 10px;
  border: 1px solid rgba(216, 230, 251, 0.95);
  background: #fff;
  box-shadow: 0 4px 12px rgba(15, 46, 106, 0.08);
  cursor: zoom-in;
}

.bubble-user .message-thumb {
  border-color: rgba(255, 255, 255, 0.56);
  box-shadow: 0 5px 14px rgba(15, 46, 106, 0.18);
}

.message-thumb :deep(img) {
  display: block;
}

.message-body.is-code pre,
.message-aux pre {
  margin: 0;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
}

.message-body.is-code pre {
  padding: 10px 12px;
  border-radius: 10px;
  background: #0f172a;
  color: #e2e8f0;
}

.file-change-card {
  padding: 0;
  overflow: hidden;
}

.file-change-card details {
  width: 100%;
}

.file-change-summary {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 48px;
  padding: 12px 14px;
  cursor: pointer;
  font-size: 13px;
  color: var(--cf-text-secondary);
}

.file-change-title {
  color: var(--cf-text-heavy);
  font-weight: 700;
}

.file-change-action {
  margin-left: auto;
  color: var(--cf-primary-dark);
  font-size: 12px;
  font-weight: 600;
}

.file-change-list {
  display: flex;
  flex-direction: column;
  gap: 0;
  padding: 2px 14px 10px;
  border-top: 1px solid rgba(216, 230, 251, 0.9);
}

.file-change-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  padding: 7px 0;
  color: var(--cf-text-secondary);
  font-size: 12px;
}

.file-change-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
}

.file-change-stats {
  display: inline-flex;
  gap: 7px;
  white-space: nowrap;
}

.diff-add {
  color: #059669;
  font-weight: 650;
}

.diff-del {
  color: #dc2626;
  font-weight: 650;
}

.message-aux {
  margin-top: 10px;
}

.message-aux summary {
  cursor: pointer;
  font-size: 12px;
  color: var(--cf-text-secondary);
  margin-bottom: 8px;
}

.message-aux pre {
  max-height: 220px;
  overflow: auto;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.05);
  color: var(--cf-text-secondary);
}

.tool-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.tool-summary {
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
  gap: 8px;
}

.tool-main {
  min-width: 0;
  flex: 1;
}

.tool-name {
  font-size: 13px;
  font-weight: 700;
  color: var(--cf-text-heavy);
}

.tool-headline {
  margin-top: 2px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
  font-size: 12px;
  line-height: 1.5;
  color: var(--cf-text-secondary);
}

.tool-type {
  flex-shrink: 0;
}

.tool-command-tag {
  display: inline-block;
  min-width: 0;
  max-width: min(100%, 560px);
  padding: 1px 8px;
  border-radius: 999px;
  background: rgba(51, 136, 255, 0.08);
  border: 1px solid rgba(151, 194, 255, 0.9);
  color: var(--cf-primary-dark);
  font-size: 11px;
  line-height: 1.6;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
}

.changes-drawer :deep(.el-drawer__body) {
  padding: 0;
  min-height: 0;
}

.changes-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  min-height: 0;
  padding: 0 16px 16px;
}

.change-scope-bar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
}

.change-scope-bar :deep(.el-input) {
  grid-column: 1 / -1;
}

.turn-scope-chip {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  min-height: 32px;
  padding: 0 12px;
  border: 1px solid rgba(151, 194, 255, 0.95);
  border-radius: 6px;
  background: #eef5ff;
  color: #1d4ed8;
  font-size: 13px;
  font-weight: 650;
}

.changes-summary-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  padding: 9px 12px;
  border: 1px solid rgba(216, 230, 251, 0.95);
  border-radius: 10px;
  background: #f8fbff;
  color: var(--cf-text-secondary);
  font-size: 13px;
  font-weight: 650;
}

.changed-file-list {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  max-height: 220px;
  overflow: auto;
  border: 1px solid rgba(216, 230, 251, 0.95);
  border-radius: 10px;
}

.changed-file-row {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 7px 10px;
  border: 0;
  border-bottom: 1px solid rgba(216, 230, 251, 0.75);
  background: #fff;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.changed-file-row:last-child {
  border-bottom: 0;
}

.changed-file-row.is-selected {
  background: #eef6ff;
}

.changed-file-status {
  color: var(--cf-primary-dark);
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  font-weight: 700;
}

.changed-file-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--cf-text-heavy);
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
}

.changed-file-stats {
  display: inline-flex;
  gap: 7px;
  white-space: nowrap;
  font-size: 12px;
}

.turn-change-card {
  width: min(100%, 860px);
  align-self: flex-start;
  border: 1px solid rgba(216, 230, 251, 0.95);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.86);
  overflow: hidden;
}

.turn-change-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 12px;
  border-bottom: 1px solid rgba(216, 230, 251, 0.78);
  color: var(--cf-text-secondary);
  font-size: 13px;
  font-weight: 700;
}

.turn-change-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.turn-change-actions button {
  border: 0;
  background: transparent;
  color: var(--cf-primary-dark);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}

.turn-change-actions button:disabled {
  color: var(--cf-text-lighter);
  cursor: not-allowed;
}

.turn-change-list {
  display: flex;
  flex-direction: column;
}

.turn-change-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 7px 12px;
  border: 0;
  border-bottom: 1px solid rgba(216, 230, 251, 0.6);
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.turn-change-row:last-child {
  border-bottom: 0;
}

.turn-change-row:hover {
  background: #f8fbff;
}

.turn-change-more {
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  min-height: 34px;
  padding: 7px 12px;
  border: 0;
  background: #fff;
  color: var(--cf-text-secondary);
  font-size: 12px;
  font-weight: 650;
  cursor: pointer;
}

.turn-change-more:hover {
  color: var(--cf-primary-dark);
  background: #f8fbff;
}

.file-detail-panel {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.file-detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.file-detail-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--cf-text-heavy);
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  font-weight: 700;
}

.file-detail-code {
  flex: 1;
  min-height: 180px;
  max-height: none;
  overflow: auto;
}

.diff-viewer {
  margin: 0;
  padding: 8px 0;
  border: 1px solid rgba(216, 230, 251, 0.95);
  border-radius: 10px;
  background: #fff;
  color: #1f2937;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.55;
  overflow: auto;
}

.inline-diff-viewer {
  margin-top: 10px;
  max-height: 360px;
}

.diff-line {
  min-height: 18px;
  padding: 0 12px;
  white-space: pre;
}

.diff-line.is-add {
  background: #e8f7ed;
  color: #047857;
}

.diff-line.is-del {
  background: #fde8e8;
  color: #b91c1c;
}

.diff-line.is-hunk {
  background: #eff6ff;
  color: #2563eb;
  font-weight: 700;
}

.diff-line.is-file,
.diff-line.is-meta {
  background: #f8fafc;
  color: #64748b;
  font-weight: 650;
}

.diff-empty,
.review-loading {
  padding: 16px;
  color: var(--cf-text-secondary);
  font-size: 13px;
}

.review-preview {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.review-preview-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.changes-summary-row.is-review {
  flex: 1;
}

.review-file-strip {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 2px;
}

.review-file-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 260px;
  min-height: 34px;
  padding: 6px 10px;
  border: 1px solid rgba(216, 230, 251, 0.95);
  border-radius: 10px;
  background: #fff;
  color: var(--cf-text-heavy);
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  cursor: pointer;
}

.review-file-chip > span:first-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.review-file-chip.is-selected {
  border-color: rgba(51, 136, 255, 0.8);
  background: #eef6ff;
}

.review-diff-viewer {
  max-height: 420px;
}

.file-content-block {
  margin: 0;
  padding: 10px 12px;
  border-radius: 10px;
  background: #0f172a;
  color: #e2e8f0;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre;
}

.file-readable-error,
.file-truncated-note {
  padding: 10px 12px;
  border-radius: 10px;
  background: #fff8eb;
  color: #92400e;
  font-size: 13px;
}

.tool-details {
  border-top: 1px solid rgba(216, 230, 251, 0.9);
  padding-top: 10px;
}

.tool-details summary {
  cursor: pointer;
  font-size: 12px;
  color: var(--cf-text-secondary);
  font-weight: 600;
}

.tool-output {
  margin-top: 10px;
}

.tool-output-title {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 700;
  color: var(--cf-text-secondary);
}

.markdown-body :deep(*) {
  word-break: break-word;
}

.markdown-body :deep(p),
.markdown-body :deep(ul),
.markdown-body :deep(ol),
.markdown-body :deep(blockquote),
.markdown-body :deep(pre),
.markdown-body :deep(table) {
  margin: 0 0 10px;
}

.markdown-body :deep(p:last-child),
.markdown-body :deep(ul:last-child),
.markdown-body :deep(ol:last-child),
.markdown-body :deep(blockquote:last-child),
.markdown-body :deep(pre:last-child),
.markdown-body :deep(table:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 20px;
}

.markdown-body :deep(code) {
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  padding: 1px 4px;
  border-radius: 4px;
  background: rgba(15, 23, 42, 0.08);
}

.markdown-body :deep(pre) {
  overflow: auto;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.06);
}

.markdown-body :deep(img) {
  max-width: 120px;
  max-height: 120px;
  border-radius: 10px;
}

.typing-cursor {
  display: inline;
  color: var(--cf-primary);
  animation: blink-cursor 0.8s step-end infinite;
}

.new-message-pill {
  position: absolute;
  right: 18px;
  bottom: 18px;
  z-index: 4;
  border: 0;
  border-radius: 999px;
  padding: 10px 14px;
  background: #2f6fec;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  box-shadow: 0 10px 24px rgba(47, 111, 236, 0.28);
  cursor: pointer;
}

.new-message-pill-enter-active,
.new-message-pill-leave-active {
  transition: all 0.2s ease;
}

.new-message-pill-enter-from,
.new-message-pill-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@keyframes blink-cursor {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.input-area {
  padding: 10px 16px;
  background: var(--cf-card);
  border-top: 1px solid var(--cf-border-light);
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.04);
}

.input-status-hint {
  margin: 0 0 6px;
  color: var(--cf-text-secondary);
  font-size: 12px;
  line-height: 1.3;
}

.pending-image-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0 0 8px;
}

.pending-image-chip {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  max-width: 220px;
  padding: 4px 8px;
  border: 1px solid rgba(151, 194, 255, 0.78);
  border-radius: 999px;
  background: rgba(51, 136, 255, 0.08);
  color: var(--cf-primary-dark);
  font-size: 12px;
  font-weight: 600;
}

.pending-image-chip span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pending-image-chip button {
  border: 0;
  padding: 0;
  background: transparent;
  color: var(--cf-text-secondary);
  font-size: 12px;
  cursor: pointer;
}

.hidden-file-input {
  display: none;
}

.input-row {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.input-plus-btn {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: 12px;
}

.input-row :deep(.el-textarea__inner) {
  border-radius: 12px;
  padding: 8px 12px;
  font-size: 14px;
  resize: none;
}

.send-btn {
  border-radius: 12px;
  height: 36px;
  flex-shrink: 0;
}

.session-detail-page.is-mobile {
  height: auto;
  min-height: 100%;
  overflow: visible;
}

.session-detail-page.is-mobile .session-hero {
  margin: 0 10px 8px;
  padding: 8px 10px;
  border-radius: 12px;
  overflow: hidden;
}

.session-detail-page.is-mobile .hero-top {
  align-items: center;
}

.session-detail-page.is-mobile .hero-main {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
  width: 100%;
  min-width: 0;
}

.session-detail-page.is-mobile .hero-title-group {
  width: 100%;
  min-width: 0;
  align-items: stretch;
}

.session-detail-page.is-mobile .hero-name-row {
  display: flex;
  flex-wrap: nowrap;
  justify-content: flex-start;
  align-items: center;
  width: 100%;
  max-width: 100%;
  gap: 6px;
  overflow: hidden;
}

.session-detail-page.is-mobile .hero-name-row :deep(.el-tag),
.session-detail-page.is-mobile .live-indicator {
  flex: 0 0 auto;
}

.session-detail-page.is-mobile .hero-meta-row {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  width: 100%;
  min-width: 0;
  overflow: hidden;
}

.session-detail-page.is-mobile .hero-tags {
  flex: 1 1 auto;
  min-width: 0;
  flex-wrap: nowrap;
  overflow: hidden;
}

.session-detail-page.is-mobile .hero-cwd {
  flex: 0 1 auto;
  min-width: 0;
  max-width: 100%;
  order: 0;
}

.session-detail-page.is-mobile .hero-status-card {
  flex-direction: row;
  align-items: stretch;
  min-width: 0;
  max-width: 100%;
  padding: 7px 9px;
}

.session-detail-page.is-mobile .hero-primary-actions {
  flex: 0 0 auto;
  justify-content: flex-start;
  flex-wrap: wrap;
}

.session-detail-page.is-mobile .hero-status-copy {
  min-width: 0;
  overflow: hidden;
}

.session-detail-page.is-mobile .hero-status-value,
.session-detail-page.is-mobile .hero-status-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-detail-page.is-mobile .hero-name {
  max-width: 100%;
  font-size: 18px;
}

.session-detail-page.is-mobile .hero-preview {
  max-width: 100%;
}

.session-detail-page.is-mobile .hero-status-card {
  width: 100%;
}

.session-detail-page.is-mobile .hero-status-desc {
  display: none;
}

.session-detail-page.is-mobile .content-area {
  overflow: visible;
  min-height: auto;
  padding: 0 0 0;
  background: transparent;
}

.session-detail-page.is-mobile .chat-shell {
  border-radius: 14px;
}

.session-detail-page.is-mobile .chat-area {
  padding: 10px 12px 14px;
}

.session-detail-page.is-mobile .new-message-pill {
  right: 12px;
  bottom: 12px;
}

.session-detail-page.is-mobile .message-bubble,
.session-detail-page.is-mobile .bubble-user {
  max-width: 100%;
  width: 100%;
}

.session-detail-page.is-mobile .input-area {
  position: sticky;
  bottom: 0;
  z-index: 5;
  padding: 8px 10px;
  box-shadow: 0 -6px 18px rgba(15, 46, 106, 0.08);
}

.session-detail-page.is-mobile .input-row :deep(.el-textarea__inner) {
  font-size: 16px;
}

.session-detail-page.is-mobile .approval-card {
  flex-direction: column;
  align-items: flex-start;
}

.session-detail-page.is-mobile .approval-actions {
  margin-top: 6px;
  width: 100%;
  justify-content: flex-end;
}

.session-detail-page.is-mobile .changes-panel {
  padding: 0 12px 12px;
}

.session-detail-page.is-mobile .change-scope-bar {
  grid-template-columns: 1fr auto;
}

.session-detail-page.is-mobile .changed-file-list {
  max-height: 30vh;
}

.session-detail-page.is-mobile .file-detail-head {
  align-items: stretch;
  flex-direction: column;
}

.session-detail-page.is-mobile .file-detail-code {
  min-height: 240px;
}
</style>
