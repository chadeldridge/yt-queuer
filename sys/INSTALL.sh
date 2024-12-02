#!/usr/bin/env bash

ORIG_USER="ytqueuer"
ORIG_HOME="/opt/ytqueuer"
USER="ytqueuer"
HOME="/opt/ytqueuer"
SERVICE="ytqueuer.service"
LOGROTATE="ytqueuer.logrotate"

update_user() {
        local u=$1
        if [[ $u == "" ]]; then
                echo "user cannot be empty"
                help
                exit 1
        fi

        if [[ $u == "root" ]]; then
                echo "user cannot be root"
                exit 1
        fi

        if [[ $u != $ORIG_USER ]]; then
                USER=$u
        fi
}

update_home() {
        local h="$1"
        if [[ $h == "" ]]; then
                echo "home directory cannot be empty"
                help
                exit 1
        fi

        if [[ $h == "/" ]]; then
                echo "home directory cannot be /"
                help
                exit 1
        fi

        f=${h:0:1}
        echo "f: $f"
        if [[ $f != "/" ]]; then
                echo "use full path to home directory"
                echo "e.g. /opt/ytqueuer"
                help
                exit 1
        fi

        BASEDIR=$(echo $h | cut -d'/' -f2)
        if [[ "$BASEDIR" == "root" ]]; then
                echo "home directory cannot be in /root"
                help
                exit 1
        fi

        if [[ $h != $ORIG_HOME ]]; then
                HOME=$h
        fi
}

help() {
        echo "Usage: $0 [-u user] [-h home]"
        echo "  -u user: User to create. Default: ${ORIG_USER}"
        echo "  -h home: Full path of the desired home directory for ytqueuer. Default: ${ORIG_HOME}"
        exit 1
}

# Exit if we are not running as root.
if [ $(id -u) -ne 0 ]; then
        echo "must be run as root, use sudo"
        help
        exit 1
fi

while getopts 'u:h:s:' flag; do
                case "${flag}" in
                u) update_user "${OPTARG}" ;;
                h) update_home "${OPTARG}" ;;
                *) echo "unexpected option ${flag}"
                        help
                        exit 1;;
        esac
done

if [[ $USER != $ORIG_USER ]]; then
        echo "sed -i 's#=$ORIG_USER#=$USER#g' $SERVICE"
        echo "sed -i 's#$ORIG_USER $ORIG_USER#$USER $USER#g' $LOGROTATE"
fi

if [[ $HOME != $ORIG_HOME ]]; then
        echo "sed -i 's#$ORIG_HOME#$HOME#g' $SERVICE"
        echo "sed -i 's#$ORIG_HOME#$HOME#g' $LOGROTATE"
fi

# Create user.
if id -u $USER &>/dev/null; then
        echo "skiping user setup, user $USER already exists"
else
        if grep -q "^video:" /etc/group; then
                useradd -r -s /bin/false -m -d $HOME -c 'yt-queuer Service User' $USER -G video || exit 1
        else
                useradd -r -s /bin/false -m -d $HOME -c 'yt-queuer Service User' $USER || exit 1
        fi
fi

# Copy files.
if [ ! -d $HOME ]; then
        mkdir -p $HOME
fi

cp -r . $HOME
cd $HOME

# Set permissions.
chown -R $USER:$USER $HOME

# Add add user to sudoers.
if [ -d /etc/sudoers.d ]; then
        if [ ! -f /etc/sudoers.d/$USER ]; then
                echo "$USER ALL=(ALL) NOPASSWD: $HOME/wol" > /etc/sudoers.d/$USER
        else
                grep -q "$USER .+? NOPASSWD: .*?$HOME/wol" /etc/sudoers.d/$USER || echo "$USER ALL=(ALL) NOPASSWD: $HOME/wol" >> /etc/sudoers.d/$USER
        fi
else
        grep -qxF "$USER .+? NOPASSWD: .*?$HOME/wol" /etc/sudoers || echo "$USER ALL=(ALL) NOPASSWD: $HOME/wol" >> /etc/sudoers
fi

# Install logrotate.
#if [ ! -d /etc/logrotate.d ]; then
#        echo "!!! logrotate not installed, be sure to setup log rotation for ytqueuer !!!"
#else
#        mv $LOGROTATE /etc/logrotate.d/ytqueuer.conf
#fi

# Install service.
mv $SERVICE /etc/systemd/system/

# Enable and start service.
systemctl daemon-reload
systemctl enable $SERVICE

ENABLED=$(systemctl is-enabled $SERVICE | tr -d '\n')
if [ $ENABLED != "enabled" ]; then
        echo "failed to enable service"
        exit 1
fi

echo "ytqueuer installed successfully as ${USER} in ${HOME}"
echo "starting ytqueuer service"
systemctl start $SERVICE