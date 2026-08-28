-- ==================== USERS ====================
INSERT INTO "users" ("id", "username", "password_hash", "role", "created_at", "updated_at") VALUES
  ('a0000000-0000-0000-0000-000000000001', 'carlos.admin', '$2a$10$xJwL5v5Jz5U5Z5Z5Z5Z5ZuZ5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5', 'ADMIN', NOW(), NOW()),
  ('a0000000-0000-0000-0000-000000000002', 'maria.atendente', '$2a$10$xJwL5v5Jz5U5Z5Z5Z5Z5ZuZ5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5', 'ATENDENTE', NOW(), NOW()),
  ('a0000000-0000-0000-0000-000000000003', 'joao.mecanico', '$2a$10$xJwL5v5Jz5U5Z5Z5Z5Z5ZuZ5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5', 'MECANICO', NOW(), NOW()),
  ('a0000000-0000-0000-0000-000000000004', 'ana.mecanica', '$2a$10$xJwL5v5Jz5U5Z5Z5Z5Z5ZuZ5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5', 'MECANICO', NOW(), NOW()),
  ('a0000000-0000-0000-0000-000000000005', 'pedro.estoque', '$2a$10$xJwL5v5Jz5U5Z5Z5Z5Z5ZuZ5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5', 'CONTROLADOR_ESTOQUE', NOW(), NOW()),
  ('a0000000-0000-0000-0000-000000000006', 'dev', '$2a$10$xJwL5v5Jz5U5Z5Z5Z5Z5ZuZ5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5', 'admin', NOW(), NOW());

-- ==================== CUSTOMERS ====================
INSERT INTO "customers" ("id", "name", "document", "document_type", "phone", "email", "created_at", "updated_at") VALUES
  ('b0000000-0000-0000-0000-000000000001', 'Roberto Silva', '12345678901', 'CPF', '11999990001', 'roberto.silva@email.com', NOW(), NOW()),
  ('b0000000-0000-0000-0000-000000000002', 'Fernanda Oliveira', '98765432100', 'CPF', '11999990002', 'fernanda.oliveira@email.com', NOW(), NOW()),
  ('b0000000-0000-0000-0000-000000000003', 'Transportes Rápido LTDA', '12345678000199', 'CNPJ', '1133330001', 'contato@transportesrapido.com', NOW(), NOW()),
  ('b0000000-0000-0000-0000-000000000004', 'Lucas Mendes', '11122233344', 'CPF', '11999990004', 'lucas.mendes@email.com', NOW(), NOW()),
  ('b0000000-0000-0000-0000-000000000005', 'Auto Peças Centro LTDA', '98765432000188', 'CNPJ', '1133330005', 'contato@autopecascentro.com', NOW(), NOW());

-- ==================== VEHICLES ====================
INSERT INTO "vehicles" ("id", "customer_id", "license_plate", "brand", "model", "year", "created_at", "updated_at") VALUES
  ('c0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001', 'ABC1D23', 'Toyota', 'Corolla', 2022, NOW(), NOW()),
  ('c0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000001', 'XYZ9K88', 'Honda', 'Civic', 2021, NOW(), NOW()),
  ('c0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000002', 'DEF4G56', 'Volkswagen', 'Golf', 2023, NOW(), NOW()),
  ('c0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000003', 'GHI7H90', 'Ford', 'Cargo 816', 2020, NOW(), NOW()),
  ('c0000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000004', 'JKL2M34', 'Chevrolet', 'Onix', 2024, NOW(), NOW());

-- ==================== SERVICES ====================
INSERT INTO "services" ("id", "title", "description", "price_cents", "estimated_time_minutes", "active", "created_at", "updated_at") VALUES
  ('d0000000-0000-0000-0000-000000000001', 'Troca de Óleo', 'Troca de óleo do motor com filtro', 15000, 30, true, NOW(), NOW()),
  ('d0000000-0000-0000-0000-000000000002', 'Alinhamento e Balanceamento', 'Alinhamento de direção e balanceamento das 4 rodas', 18000, 60, true, NOW(), NOW()),
  ('d0000000-0000-0000-0000-000000000003', 'Troca de Pastilhas de Freio', 'Substituição das pastilhas de freio dianteiras', 25000, 45, true, NOW(), NOW()),
  ('d0000000-0000-0000-0000-000000000004', 'Revisão Completa 10.000km', 'Revisão completa incluindo fluidos, filtros e inspeção geral', 45000, 120, true, NOW(), NOW()),
  ('d0000000-0000-0000-0000-000000000005', 'Troca de Correia Dentada', 'Substituição da correia dentada e tensor', 60000, 180, true, NOW(), NOW()),
  ('d0000000-0000-0000-0000-000000000006', 'Diagnóstico Eletrônico', 'Leitura e diagnóstico via scanner OBD2', 8000, 30, true, NOW(), NOW());

-- ==================== ITEMS ====================
INSERT INTO "supplies" ("id", "title", "type", "price_cents", "stock_quantity", "minimum_stock", "active", "created_at", "updated_at") VALUES
  ('e0000000-0000-0000-0000-000000000001', 'Óleo Motor 5W30 (litro)', 'INSUMO', 4500, 50, 10, true, NOW(), NOW()),
  ('e0000000-0000-0000-0000-000000000002', 'Filtro de Óleo Universal', 'PECA', 3500, 30, 5, true, NOW(), NOW()),
  ('e0000000-0000-0000-0000-000000000003', 'Pastilha de Freio Dianteira (jogo)', 'PECA', 12000, 15, 3, true, NOW(), NOW()),
  ('e0000000-0000-0000-0000-000000000004', 'Correia Dentada', 'PECA', 18000, 8, 2, true, NOW(), NOW()),
  ('e0000000-0000-0000-0000-000000000005', 'Tensor da Correia', 'PECA', 9500, 6, 2, true, NOW(), NOW()),
  ('e0000000-0000-0000-0000-000000000006', 'Filtro de Ar', 'PECA', 5000, 20, 5, true, NOW(), NOW()),
  ('e0000000-0000-0000-0000-000000000007', 'Filtro de Combustível', 'PECA', 6000, 18, 4, true, NOW(), NOW()),
  ('e0000000-0000-0000-0000-000000000008', 'Fluido de Freio DOT4 (500ml)', 'INSUMO', 3000, 25, 5, true, NOW(), NOW()),
  ('e0000000-0000-0000-0000-000000000009', 'Peso de Balanceamento (unidade)', 'INSUMO', 200, 200, 50, true, NOW(), NOW()),
  ('e0000000-0000-0000-0000-000000000010', 'Líquido de Arrefecimento (litro)', 'INSUMO', 2500, 30, 8, true, NOW(), NOW());

-- ==================== WORK_ORDERS ====================
INSERT INTO "work_orders" ("id", "code", "title", "description", "customer_id", "vehicle_id", "opened_by_user_id", "assigned_technician_id", "status", "total_estimated_price_cents", "received_at", "quote_sent_at", "approved_at", "started_at", "finished_at", "delivered_at", "created_at", "updated_at") VALUES
  ('10000000-0000-0000-0000-000000000001', 'OS-2026-0001', 'Revisão Corolla 2022', 'Cliente solicitou revisão completa de 10.000km', 'b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000003', 'EM_EXECUCAO', 63000, NOW() - INTERVAL '3 days', NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', NULL, NULL, NOW() - INTERVAL '3 days', NOW()),
  ('10000000-0000-0000-0000-000000000002', 'OS-2026-0002', 'Freio Golf 2023', 'Barulho ao frear, possível desgaste de pastilha', 'b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000004', 'AGUARDANDO_APROVACAO', 28000, NOW() - INTERVAL '1 day', NOW(), NULL, NULL, NULL, NULL, NOW() - INTERVAL '1 day', NOW()),
  ('10000000-0000-0000-0000-000000000003', 'OS-2026-0003', 'Diagnóstico Onix 2024', 'Luz de check engine acesa', 'b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000002', NULL, 'RECEBIDA', 8000, NOW(), NULL, NULL, NULL, NULL, NULL, NOW(), NOW()),
  ('10000000-0000-0000-0000-000000000004', 'OS-2026-0004', 'Troca Correia Cargo', 'Manutenção preventiva da correia dentada', 'b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000003', 'FINALIZADA', 60000, NOW() - INTERVAL '7 days', NOW() - INTERVAL '6 days', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days', NOW() - INTERVAL '3 days', NULL, NOW() - INTERVAL '7 days', NOW()),
  ('10000000-0000-0000-0000-000000000005', 'OS-2026-0005', 'Óleo e Alinhamento Civic', 'Troca de óleo + alinhamento e balanceamento', 'b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000004', 'ENTREGUE', 33000, NOW() - INTERVAL '10 days', NOW() - INTERVAL '9 days', NOW() - INTERVAL '8 days', NOW() - INTERVAL '8 days', NOW() - INTERVAL '7 days', NOW() - INTERVAL '6 days', NOW() - INTERVAL '10 days', NOW());

-- ==================== WORK_ORDER_SERVICES ====================
INSERT INTO "work_order_services" ("id", "work_order_id", "service_id", "service_title_snapshot", "service_description_snapshot", "service_price_cents_snapshot", "service_estimated_time_minutes_snapshot", "approval_status", "status", "started_at", "finished_at", "created_at", "updated_at") VALUES
  ('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000004', 'Revisão Completa 10.000km', 'Revisão completa incluindo fluidos, filtros e inspeção geral', 45000, 120, 'APROVADO', 'EM_EXECUCAO', NOW() - INTERVAL '1 day', NULL, NOW() - INTERVAL '3 days', NOW()),
  ('20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 'Alinhamento e Balanceamento', 'Alinhamento de direção e balanceamento das 4 rodas', 18000, 60, 'APROVADO', 'PENDENTE', NULL, NULL, NOW() - INTERVAL '3 days', NOW()),
  ('20000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000003', 'Troca de Pastilhas de Freio', 'Substituição das pastilhas de freio dianteiras', 25000, 45, 'PENDENTE', 'AGUARDANDO_APROVACAO', NULL, NULL, NOW() - INTERVAL '1 day', NOW()),
  ('20000000-0000-0000-0000-000000000004', '10000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000006', 'Diagnóstico Eletrônico', 'Leitura e diagnóstico via scanner OBD2', 8000, 30, 'PENDENTE', 'PENDENTE', NULL, NULL, NOW() - INTERVAL '1 day', NOW()),
  ('20000000-0000-0000-0000-000000000005', '10000000-0000-0000-0000-000000000003', 'd0000000-0000-0000-0000-000000000006', 'Diagnóstico Eletrônico', 'Leitura e diagnóstico via scanner OBD2', 8000, 30, 'PENDENTE', 'PENDENTE', NULL, NULL, NOW(), NOW()),
  ('20000000-0000-0000-0000-000000000006', '10000000-0000-0000-0000-000000000004', 'd0000000-0000-0000-0000-000000000005', 'Troca de Correia Dentada', 'Substituição da correia dentada e tensor', 60000, 180, 'APROVADO', 'FINALIZADO', NOW() - INTERVAL '3 days 3 hours 10 minutes', NOW() - INTERVAL '3 days', NOW() - INTERVAL '7 days', NOW()),
  ('20000000-0000-0000-0000-000000000007', '10000000-0000-0000-0000-000000000005', 'd0000000-0000-0000-0000-000000000001', 'Troca de Óleo', 'Troca de óleo do motor com filtro', 15000, 30, 'APROVADO', 'FINALIZADO', NOW() - INTERVAL '7 days 28 minutes', NOW() - INTERVAL '7 days', NOW() - INTERVAL '10 days', NOW()),
  ('20000000-0000-0000-0000-000000000008', '10000000-0000-0000-0000-000000000005', 'd0000000-0000-0000-0000-000000000002', 'Alinhamento e Balanceamento', 'Alinhamento de direção e balanceamento das 4 rodas', 18000, 60, 'APROVADO', 'FINALIZADO', NOW() - INTERVAL '7 days 1 hour 5 minutes', NOW() - INTERVAL '7 days', NOW() - INTERVAL '10 days', NOW());

-- ==================== WORK_ORDER_SERVICE_ITEMS ====================
INSERT INTO "work_order_service_supplies" ("id", "work_order_service_id", "supply_id", "supply_title_snapshot", "supply_price_cents_snapshot", "supply_quantity", "created_at", "updated_at") VALUES
  ('30000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000001', 'Óleo Motor 5W30 (litro)', 4500, 4, NOW() - INTERVAL '3 days', NOW()),
  ('30000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000002', 'Filtro de Óleo Universal', 3500, 1, NOW() - INTERVAL '3 days', NOW()),
  ('30000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000006', 'Filtro de Ar', 5000, 1, NOW() - INTERVAL '3 days', NOW()),
  ('30000000-0000-0000-0000-000000000004', '20000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000007', 'Filtro de Combustível', 6000, 1, NOW() - INTERVAL '3 days', NOW()),
  ('30000000-0000-0000-0000-000000000005', '20000000-0000-0000-0000-000000000002', 'e0000000-0000-0000-0000-000000000009', 'Peso de Balanceamento (unidade)', 200, 16, NOW() - INTERVAL '3 days', NOW()),
  ('30000000-0000-0000-0000-000000000006', '20000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000003', 'Pastilha de Freio Dianteira (jogo)', 12000, 1, NOW() - INTERVAL '1 day', NOW()),
  ('30000000-0000-0000-0000-000000000007', '20000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000008', 'Fluido de Freio DOT4 (500ml)', 3000, 1, NOW() - INTERVAL '1 day', NOW()),
  ('30000000-0000-0000-0000-000000000008', '20000000-0000-0000-0000-000000000006', 'e0000000-0000-0000-0000-000000000004', 'Correia Dentada', 18000, 1, NOW() - INTERVAL '7 days', NOW()),
  ('30000000-0000-0000-0000-000000000009', '20000000-0000-0000-0000-000000000006', 'e0000000-0000-0000-0000-000000000005', 'Tensor da Correia', 9500, 1, NOW() - INTERVAL '7 days', NOW()),
  ('30000000-0000-0000-0000-000000000010', '20000000-0000-0000-0000-000000000007', 'e0000000-0000-0000-0000-000000000001', 'Óleo Motor 5W30 (litro)', 4500, 4, NOW() - INTERVAL '10 days', NOW()),
  ('30000000-0000-0000-0000-000000000011', '20000000-0000-0000-0000-000000000007', 'e0000000-0000-0000-0000-000000000002', 'Filtro de Óleo Universal', 3500, 1, NOW() - INTERVAL '10 days', NOW()),
  ('30000000-0000-0000-0000-000000000012', '20000000-0000-0000-0000-000000000008', 'e0000000-0000-0000-0000-000000000009', 'Peso de Balanceamento (unidade)', 200, 16, NOW() - INTERVAL '10 days', NOW());

