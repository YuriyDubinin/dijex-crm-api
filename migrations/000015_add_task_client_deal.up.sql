-- Прямые связи задачи с клиентом и сделкой (1:N): задача может относиться к
-- конкретному клиенту и/или сделке. ON DELETE SET NULL — при удалении клиента/
-- сделки задача не удаляется, ссылка просто обнуляется.
ALTER TABLE tasks
    ADD COLUMN client_id UUID REFERENCES clients(id) ON DELETE SET NULL,
    ADD COLUMN deal_id   UUID REFERENCES deals(id) ON DELETE SET NULL;

CREATE INDEX idx_tasks_client ON tasks (client_id);
CREATE INDEX idx_tasks_deal ON tasks (deal_id);
