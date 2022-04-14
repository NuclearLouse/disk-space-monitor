#!/bin/bash
# git clone https://redits.oculeus.com/asorokin/disk-usage-monitor_bin.git disk-usage-monitor
CURRENT=$(df / | grep / | awk '{ print $5}' | sed 's/%//g')
THRESHOLD=90
if [ "$CURRENT" -gt "$THRESHOLD" ] ; then
mail -s 'Заканчивается дисковое пространство' -r admin@example.com << EOF
В вашем корневом разделе сервера server1, осталось слишком мало дискового пространства. Используется: $CURRENT%
EOF
fi
