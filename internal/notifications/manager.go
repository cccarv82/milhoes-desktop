package notifications

import (
	"fmt"
	"lottery-optimizer-gui/internal/logs"
	"time"
)

// NotificationManager gerencia todas as notificações do app
type NotificationManager struct {
	enabled       bool
	emailEnabled  bool
	pushEnabled   bool
	reminderTime  time.Duration
	notifications []Notification
}

// Notification representa uma notificação
type Notification struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // "reminder", "result", "performance", "system"
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Priority  string                 `json:"priority"` // "low", "medium", "high", "urgent"
	Category  string                 `json:"category"` // "game", "finance", "system", "achievement"
	CreatedAt time.Time              `json:"createdAt"`
	ReadAt    *time.Time             `json:"readAt,omitempty"`
	ActionURL string                 `json:"actionUrl,omitempty"`
	Icon      string                 `json:"icon,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// NotificationSettings configura o comportamento das notificações
type NotificationSettings struct {
	Enabled           bool            `json:"enabled"`
	PushEnabled       bool            `json:"pushEnabled"`
	EmailEnabled      bool            `json:"emailEnabled"`
	ReminderTime      time.Duration   `json:"reminderTime"`
	QuietHours        QuietHours      `json:"quietHours"`
	Categories        map[string]bool `json:"categories"`
	MinPrizeAlert     float64         `json:"minPrizeAlert"`
	PerformanceAlerts bool            `json:"performanceAlerts"`
}

// QuietHours define horários de silêncio
type QuietHours struct {
	Enabled bool `json:"enabled"`
	Start   int  `json:"start"` // Hora (0-23)
	End     int  `json:"end"`   // Hora (0-23)
}

// GlobalNotificationManager instância global
var GlobalNotificationManager *NotificationManager

// InitNotificationManager inicializa o sistema de notificações
func InitNotificationManager() {
	GlobalNotificationManager = &NotificationManager{
		enabled:       true,
		emailEnabled:  false,
		pushEnabled:   true,
		reminderTime:  time.Hour * 2, // Lembrar a cada 2 horas
		notifications: make([]Notification, 0),
	}

	logs.LogMain("🔔 Sistema de notificações inicializado")
}

// SendNotification envia uma nova notificação
func (nm *NotificationManager) SendNotification(notification Notification) error {
	if !nm.enabled {
		return nil // Notificações desabilitadas
	}

	// Adicionar timestamp
	notification.CreatedAt = time.Now()

	// Gerar ID se não fornecido
	if notification.ID == "" {
		notification.ID = fmt.Sprintf("notif_%d", time.Now().UnixNano())
	}

	// Adicionar à lista
	nm.notifications = append(nm.notifications, notification)

	// Log da notificação
	logs.LogMain("🔔 Nova notificação: %s - %s", notification.Type, notification.Title)

	// Processar baseado no tipo
	switch notification.Type {
	case "reminder":
		return nm.processReminderNotification(notification)
	case "result":
		return nm.processResultNotification(notification)
	case "performance":
		return nm.processPerformanceNotification(notification)
	case "achievement":
		return nm.processAchievementNotification(notification)
	default:
		return nm.processSystemNotification(notification)
	}
}

// processReminderNotification processa notificações de lembrete
func (nm *NotificationManager) processReminderNotification(notification Notification) error {
	// Para lembretes, exibir imediatamente se não estiver em horário de silêncio
	if nm.isQuietTime() {
		logs.LogMain("🔇 Notificação adiada - horário de silêncio")
		return nil
	}

	// Mostrar notificação push
	if nm.pushEnabled {
		nm.showPushNotification(notification)
	}

	return nil
}

// processResultNotification processa notificações de resultado
func (nm *NotificationManager) processResultNotification(notification Notification) error {
	// Resultados são sempre importantes
	if nm.pushEnabled {
		nm.showPushNotification(notification)
	}

	// Se ganhou prêmio alto, enviar email também
	if nm.emailEnabled && notification.Data != nil {
		if prize, ok := notification.Data["prize"].(float64); ok && prize > 100 {
			nm.sendEmailNotification(notification)
		}
	}

	return nil
}

// processPerformanceNotification processa notificações de performance
func (nm *NotificationManager) processPerformanceNotification(notification Notification) error {
	// Performance alerts são opcionais
	if nm.pushEnabled {
		nm.showPushNotification(notification)
	}

	return nil
}

// processAchievementNotification processa notificações de conquistas
func (nm *NotificationManager) processAchievementNotification(notification Notification) error {
	// Conquistas são sempre celebradas!
	if nm.pushEnabled {
		nm.showPushNotification(notification)
	}

	return nil
}

// processSystemNotification processa notificações do sistema
func (nm *NotificationManager) processSystemNotification(notification Notification) error {
	if nm.pushEnabled && notification.Priority == "high" {
		nm.showPushNotification(notification)
	}

	return nil
}

// showPushNotification exibe notificação push (local)
func (nm *NotificationManager) showPushNotification(notification Notification) {
	logs.LogMain("🔔 Push: %s", notification.Title)
	// TODO: Implementar notificação real do sistema operacional
}

// sendEmailNotification envia notificação por email
func (nm *NotificationManager) sendEmailNotification(notification Notification) {
	logs.LogMain("📧 Email: %s", notification.Title)
	// TODO: Implementar envio de email
}

// isQuietTime verifica se está em horário de silêncio
func (nm *NotificationManager) isQuietTime() bool {
	// Por enquanto, sempre false - implementar configuração depois
	return false
}

// GetNotifications retorna todas as notificações
func (nm *NotificationManager) GetNotifications(limit int, onlyUnread bool) []Notification {
	result := make([]Notification, 0)

	for i := len(nm.notifications) - 1; i >= 0 && len(result) < limit; i-- {
		notification := nm.notifications[i]

		if onlyUnread && notification.ReadAt != nil {
			continue
		}

		result = append(result, notification)
	}

	return result
}

// MarkAsRead marca notificação como lida
func (nm *NotificationManager) MarkAsRead(notificationID string) error {
	for i := range nm.notifications {
		if nm.notifications[i].ID == notificationID {
			now := time.Now()
			nm.notifications[i].ReadAt = &now
			return nil
		}
	}

	return fmt.Errorf("notificação não encontrada: %s", notificationID)
}

// ClearNotifications remove notificações antigas
func (nm *NotificationManager) ClearNotifications(olderThan time.Duration) int {
	cutoff := time.Now().Add(-olderThan)
	remaining := make([]Notification, 0)
	removed := 0

	for _, notification := range nm.notifications {
		if notification.CreatedAt.After(cutoff) {
			remaining = append(remaining, notification)
		} else {
			removed++
		}
	}

	nm.notifications = remaining
	logs.LogMain("🗑️ %d notificações antigas removidas", removed)

	return removed
}

// ===============================
// NOTIFICAÇÕES ESPECÍFICAS
// ===============================

// NotifyGameReminder envia lembrete de jogo
func NotifyGameReminder(lotteryType string, drawDate string) {
	if GlobalNotificationManager == nil {
		return
	}

	notification := Notification{
		Type:     "reminder",
		Title:    fmt.Sprintf("Lembrete: %s", lotteryType),
		Message:  fmt.Sprintf("Sorteio da %s hoje (%s). Não esqueça de conferir seus jogos!", lotteryType, drawDate),
		Priority: "medium",
		Category: "game",
		Icon:     "🎱",
		Data: map[string]interface{}{
			"lotteryType": lotteryType,
			"drawDate":    drawDate,
		},
	}

	GlobalNotificationManager.SendNotification(notification)
}

// NotifyGameResult notifica resultado de jogo
func NotifyGameResult(lotteryType string, won bool, prize float64, hits int) {
	if GlobalNotificationManager == nil {
		return
	}

	var title, message, icon string
	var priority string = "medium"

	if won {
		title = "🎉 Parabéns! Você Ganhou!"
		message = fmt.Sprintf("%s: %d acertos - R$ %.2f", lotteryType, hits, prize)
		icon = "🏆"
		priority = "high"
	} else {
		title = fmt.Sprintf("%s - Resultado", lotteryType)
		message = fmt.Sprintf("%d acertos. Continue tentando!", hits)
		icon = "🎱"
	}

	notification := Notification{
		Type:     "result",
		Title:    title,
		Message:  message,
		Priority: priority,
		Category: "game",
		Icon:     icon,
		Data: map[string]interface{}{
			"lotteryType": lotteryType,
			"won":         won,
			"prize":       prize,
			"hits":        hits,
		},
	}

	GlobalNotificationManager.SendNotification(notification)
}

// NotifyPerformanceAlert notifica alertas de performance
func NotifyPerformanceAlert(alertType string, value float64, description string) {
	if GlobalNotificationManager == nil {
		return
	}

	var title, icon string
	var priority string = "medium"

	switch alertType {
	case "roi_positive":
		title = "📈 ROI Positivo!"
		icon = "📈"
		priority = "high"
	case "roi_negative":
		title = "📉 ROI em Queda"
		icon = "⚠️"
	case "win_streak":
		title = "🔥 Sequência de Vitórias!"
		icon = "🔥"
		priority = "high"
	case "loss_streak":
		title = "😔 Sequência de Derrotas"
		icon = "⚡"
	default:
		title = "📊 Alerta de Performance"
		icon = "📊"
	}

	notification := Notification{
		Type:     "performance",
		Title:    title,
		Message:  description,
		Priority: priority,
		Category: "finance",
		Icon:     icon,
		Data: map[string]interface{}{
			"alertType": alertType,
			"value":     value,
		},
	}

	GlobalNotificationManager.SendNotification(notification)
}

// NotifyAchievement notifica conquistas
func NotifyAchievement(achievement string, description string) {
	if GlobalNotificationManager == nil {
		return
	}

	notification := Notification{
		Type:     "achievement",
		Title:    fmt.Sprintf("🏆 Conquista: %s", achievement),
		Message:  description,
		Priority: "high",
		Category: "achievement",
		Icon:     "🏆",
		Data: map[string]interface{}{
			"achievement": achievement,
		},
	}

	GlobalNotificationManager.SendNotification(notification)
}

// NotifySystemUpdate notifica atualizações do sistema
func NotifySystemUpdate(version string, available bool) {
	if GlobalNotificationManager == nil {
		return
	}

	var title, message string

	if available {
		title = "🚀 Atualização Disponível"
		message = fmt.Sprintf("Nova versão %s disponível! Clique para atualizar.", version)
	} else {
		title = "✅ App Atualizado"
		message = fmt.Sprintf("App atualizado para versão %s com sucesso!", version)
	}

	notification := Notification{
		Type:     "system",
		Title:    title,
		Message:  message,
		Priority: "medium",
		Category: "system",
		Icon:     "🚀",
		Data: map[string]interface{}{
			"version":   version,
			"available": available,
		},
	}

	GlobalNotificationManager.SendNotification(notification)
}
