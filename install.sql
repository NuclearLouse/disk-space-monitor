-- # git clone https://redits.oculeus.com/asorokin/disk-usage-monitor_bin.git disk-usage-monitor

-- установка при уже существующем logs-manager

psql -d captdb -U postgres

SELECT server_name FROM logs_manager.agent_bit_control;

--при раздельных серверах main и rater если надо добавить еще оин сервер на котором будет запущен агент
INSERT INTO logs_manager.agent_bit_control (server_name) VALUES ('Benin-SMS');

--tags:
-- main-disk-usage-all-log main-disk-usage-warn-log
-- rater-disk-usage-all-log rater-disk-usage-warn-log

INSERT INTO logs_manager.served_application (server_name, service_name, service_tag_general, service_tags_alert, path_to_logfile, path_to_errlogfile, keywords_alert) 
VALUES ('Benin-SMS', 'Disk Space Usage Monitor', 'sms-disk-usage-all-log', '{sms-disk-usage-warn-log}', '/home/disk-usage-monitor/disk-usage-monitor.log', NULL, '{WARNING}');

INSERT INTO logs_manager.alert_settings (id_served_app, urgent_if_message_contains, email_to) 
VALUES ((SELECT id_served_app FROM logs_manager.served_application WHERE service_name='Disk Space Usage Monitor' AND server_name='Benin-SMS'),'{threshold exceeded}','{alert@oculeus.zohodesk.eu}');

UPDATE logs_manager.agent_bit_control SET reload_config=TRUE WHERE server_name='Benin-SMS';
UPDATE logs_manager.alert_email_control SET reload_config=TRUE;

