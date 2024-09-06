import os
import platform
import subprocess
import tarfile


def get_platform():
    system = platform.system().lower()
    machine = platform.machine().lower()

    if system == 'darwin':
        return 'macos'
    elif system == 'linux':
        if 'arm' in machine or 'aarch64' in machine:
            return 'linux-arm64'
        elif 'x86_64' in machine:
            return 'linux-amd64'
    
    raise ValueError(f"Unsupported platform: {system} {machine}")

def download_file(url):
    subprocess.run(['wget', url], check=True)

def extract_tar(filename):
    with tarfile.open(filename, 'r:gz') as tar:
        tar.extractall()

def main():
    try:
        # 1. Detect OS platform
        platform = get_platform()
        
        # 2. Download the correct file
        base_url = "https://mailsherpa.sh"
        filename = f"mailsherpa-{platform}.tar.gz"
        url = f"{base_url}/{filename}"
        
        print(f"Downloading {url}")
        download_file(url)
        
        # 3. Extract the binary
        print(f"Extracting {filename}")
        extract_tar(filename)
        
        # 4. Rename the binary
        original_name = f"mailsherpa-{platform}"
        new_name = "mailsherpa"
        print(f"Renaming {original_name} to {new_name}")
        os.rename(original_name, new_name)
        
        # 5. Remove the tarball
        print(f"Removing {filename}")
        os.remove(filename)
        
        print("Operation completed successfully")
    
    except Exception as e:
        print(f"An error occurred: {e}")

if __name__ == "__main__":
    main()
