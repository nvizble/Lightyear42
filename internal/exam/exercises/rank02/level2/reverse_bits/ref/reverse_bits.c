unsigned char	reverse_bits(unsigned char octet)
{
	unsigned char	result;
	int				i;

	result = 0;
	i = 0;
	while (i < 8)
	{
		result = (result << 1) | (octet & 1);
		octet >>= 1;
		i++;
	}
	return (result);
}
