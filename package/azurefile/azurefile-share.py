#!/usr/bin/env python
# Simple script to create and delete shares
# Trying to do this with curl == (╯°□°）╯︵ ┻━┻ - Jason Greathouse (jgreat)

import os
import argparse
import re
import sys
from azure.storage.file import FileService

parser = argparse.ArgumentParser(description='Create or Delete a File Share')
parser.add_argument('command', nargs=1, help='<create|delete>')
parser.add_argument('share', nargs=1, help='<share>')

opts = parser.parse_args()
command = opts.command[0]
share = opts.share[0]

if re.match(command, '(create|delete)'):
    print('invaid command argument')
    sys.exit(1)

if not share:
    print('no share argument')
    sys.exit(1)

account_name = os.getenv('AZURE_STORAGE_ACCOUNT')
account_key = os.getenv('AZURE_STORAGE_ACCOUNT_KEY')

file_service = FileService(account_name=account_name, account_key=account_key)

if command == 'create':
    if file_service.create_share(share):
        print('Share created')
    else:
        print('Share alreaded exists')

if command == 'delete':
    if file_service.delete_share(share):
        print('Share Deleted')
    else:
        print('Share not found')
